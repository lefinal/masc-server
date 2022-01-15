package mqttbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"strings"
)

// shelliesShelly1RelayState is used for reading the relay-status for ShelliesShelly1.
type shelliesShelly1RelayState string

// shelliesShelly1RelayCommand is used for specifying the relay-command for ShelliesShelly1.
type shelliesShelly1RelayCommand string

type shelliesShelly1InputState int

// Constants for ShelliesShelly1.
const (
	shelliesShelly1RelayStateOn       shelliesShelly1RelayState         = "on"
	shelliesShelly1RelayStateOff      shelliesShelly1RelayState         = "off"
	shelliesShelly1RelayCommandOn     shelliesShelly1RelayCommand       = "on"
	shelliesShelly1RelayCommandOff    shelliesShelly1RelayCommand       = "off"
	shelliesShelly1RelayCommandToggle shelliesShelly1RelayCommand       = "toggle"
	shelliesShelly1InputStateHigh     shelliesShelly1InputState         = 1
	shelliesShelly1InputStateLow      shelliesShelly1InputState         = 0
	shelliesShelly1RelayFixtureID     messages.FixtureProviderFixtureID = "relay"
)

type shelliesShelly1State struct {
	relayState shelliesShelly1RelayState
	inputState shelliesShelly1InputState
}

// genShelliesShelly1FixtureState generates the
// messages.MessageFixtureBasicState from the given shelliesShelly1State.
func genShelliesShelly1FixtureState(state shelliesShelly1State) messages.MessageFixtureBasicState {
	return messages.MessageFixtureBasicState{
		Fixture:    shelliesShelly1RelayFixtureID,
		IsEnabled:  state.relayState == shelliesShelly1RelayStateOn,
		IsLocating: false,
	}
}

type netShelliesShelly1 struct {
	// shellyMQTTID is the device id of the shelly. Taken from topic format:
	//  shellies/shelly1-<device-id>/...
	shellyMQTTID string
	// deviceID is the actually assigned device id from MASC.
	deviceID messages.DeviceID
	// state is the known state of shelly.
	state shelliesShelly1State
	// isStateInitialized describes whether the state was set from the first fixture
	// state message. This is used in order to send an update with the initial state
	// even if the default values match the first applied fixture state.
	isStateInitialized bool
	// fixtureProviderActorID is the assigned actor id.
	fixtureProviderActorID messages.ActorID
	// No need for synchronization as we handle message after message.
}

func newNetShelliesShelly1(topic string) (*netShelliesShelly1, error) {
	// Extract shelly device id.
	topicSegments := strings.Split(topic, "/")
	if len(topicSegments) < 2 {
		return nil, errors.NewInternalError("expected at least 2 segments in topic", errors.Details{"mqtt_topic": topic})
	}
	idSegments := strings.Split(topicSegments[1], "shelly1-")
	if len(idSegments) != 2 {
		return nil, errors.NewInternalError("expected 2 id segments in topic segment", errors.Details{
			"mqtt_topic":                  topic,
			"mqtt_topic_selected_segment": topicSegments[1],
			"mqtt_id_segments":            idSegments,
		})
	}
	return &netShelliesShelly1{
		shellyMQTTID: idSegments[1],
		state: shelliesShelly1State{
			relayState: shelliesShelly1RelayStateOff,
			inputState: shelliesShelly1InputStateLow,
		},
	}, nil
}

func (shelly *netShelliesShelly1) genMessageHello() messages.MessageHello {
	return messages.MessageHello{
		Roles:           []messages.Role{messages.Role(acting.RoleTypeFixtureProvider)}, // TODO: ADD switch stuff
		SelfDescription: "shelly1",
	}
}

func (shelly *netShelliesShelly1) run(ctx context.Context, deviceID messages.DeviceID, fromMQTT chan mqtt.Message, fromMASC chan messages.MessageContainer, publisher publisher) error {
	shelly.deviceID = deviceID
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message := <-fromMQTT:
			if err := shelly.handleFromMQTT(ctx, message, publisher); err != nil {
				return errors.Wrap(err, "handle from mqtt", nil)
			}
		case message := <-fromMASC:
			if err := shelly.handleFromMASC(ctx, message, publisher); err != nil {
				return errors.Wrap(err, "handle from masc", nil)
			}
		}
	}
}

func (shelly *netShelliesShelly1) handleFromMQTT(ctx context.Context, message mqtt.Message, publisher publisher) error {
	switch {
	case strings.HasSuffix(message.Topic(), "/relay/0"):
		if err := shelly.handleRelayStateMessage(ctx, message, publisher); err != nil {
			return errors.Wrap(err, "handle relay state message", nil)
		}
	case strings.HasSuffix(message.Topic(), "/input/0"):
		if err := shelly.handleInputStateMessage(ctx, message, publisher); err != nil {
			return errors.Wrap(err, "handle input state message", nil)
		}
	}
	// Not found.
	return nil
}

func (shelly *netShelliesShelly1) handleRelayStateMessage(ctx context.Context, message mqtt.Message, publisher publisher) error {
	attempted := shelliesShelly1RelayState(message.Payload())
	if shelly.state.relayState == attempted {
		return nil
	}
	// Overwrite.
	if err := shelly.publishFixtureState(ctx, publisher); err != nil {
		return errors.Wrap(err, "overwrite fixture state", errors.Details{"attempted_relay_state": attempted})
	}
	return nil
}

func (shelly *netShelliesShelly1) handleInputStateMessage(ctx context.Context, message mqtt.Message, publisher publisher) error {
	// TODO
	return nil
}

func (shelly *netShelliesShelly1) handleFromMASC(ctx context.Context, message messages.MessageContainer, publisher publisher) error {
	// Check if we are waiting for actor assignment.
	if len(shelly.fixtureProviderActorID) == 0 {
		if message.MessageType == messages.MessageTypeYouAreIn {
			// Parse content.
			var messageContent messages.MessageYouAreIn
			if err := json.Unmarshal(message.Content, &messageContent); err != nil {
				return errors.NewJSONError(err, "unmarshal you-are-in message content", false)
			}
			// Set assigned actor id.
			shelly.fixtureProviderActorID = messageContent.ActorID
			return nil
		} else {
			logging.MQTTLogger.WithField("message_type", "unexpected message type while not being hired.")
			return nil
		}
	}
	switch message.MessageType {
	case messages.MessageTypeFixtureBasicState:
		// Parse content.
		var messageContent messages.MessageFixtureBasicState
		if err := json.Unmarshal(message.Content, &messageContent); err != nil {
			return errors.NewJSONError(err, "unmarshal fixture state", false)
		}
		// Apply.
		if err := shelly.applyFixtureState(ctx, messageContent, publisher); err != nil {
			return errors.Wrap(err, "apply fixture state", errors.Details{"fixture_state": messageContent})
		}
	case messages.MessageTypeGetFixtureOffers:
		if err := shelly.publishFixtureOffers(ctx, publisher); err != nil {
			return errors.Wrap(err, "publish fixture offers", nil)
		}
	}
	// Unknown message.
	return nil
}

func (shelly *netShelliesShelly1) applyFixtureState(ctx context.Context, fixtureState messages.MessageFixtureBasicState, publisher publisher) error {
	if shelly.isStateInitialized && genShelliesShelly1FixtureState(shelly.state) == fixtureState {
		return nil
	}
	if err := shelly.setRelayState(ctx, fixtureState.IsEnabled, publisher); err != nil {
		return errors.Wrap(err, "set relay state", errors.Details{"relay_enabled": fixtureState.IsEnabled})
	}
	return nil
}

// setRelayState sets the relay to the given enabled-state and publishes the update.
func (shelly *netShelliesShelly1) setRelayState(ctx context.Context, isEnabled bool, publisher publisher) error {
	if isEnabled {
		shelly.state.relayState = shelliesShelly1RelayStateOn
	} else {
		shelly.state.relayState = shelliesShelly1RelayStateOff
	}
	err := publisher.publishMQTT(ctx, fmt.Sprintf("shellies/shelly1-%s/relay/0/command", shelly.shellyMQTTID),
		string(shelly.state.relayState))
	if err != nil {
		return errors.Wrap(err, "publish mqtt", nil)
	}
	return nil
}

func (shelly *netShelliesShelly1) publishFixtureOffers(ctx context.Context, publisher publisher) error {
	if err := publisher.publishMASC(ctx, messages.MessageTypeFixtureOffers, shelly.fixtureProviderActorID,
		messages.MessageFixtureOffers{
			DeviceID: shelly.deviceID,
			Fixtures: []messages.OfferedFixture{{
				ProviderID: shelliesShelly1RelayFixtureID,
				Type:       messages.FixtureTypeBasic,
			}},
		}); err != nil {
		return errors.Wrap(err, "publish masc", nil)
	}
	return nil
}

func (shelly *netShelliesShelly1) publishFixtureState(ctx context.Context, publisher publisher) error {
	if len(shelly.fixtureProviderActorID) == 0 {
		logging.MQTTLogger.WithField("shelly_mqtt_id", shelly.shellyMQTTID).
			Debug("called publish fixture state with no actor id", nil)
	}
	if err := publisher.publishMASC(ctx, messages.MessageTypeFixtureBasicState, shelly.fixtureProviderActorID,
		genShelliesShelly1FixtureState(shelly.state)); err != nil {
		return errors.Wrap(err, "publish masc", nil)
	}
	return nil
}
