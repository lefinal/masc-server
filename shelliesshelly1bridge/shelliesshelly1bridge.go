package shelliesshelly1bridge

//
// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"github.com/LeFinal/masc-server/acting"
// 	"github.com/LeFinal/masc-server/errors"
// 	"github.com/LeFinal/masc-server/events"
// 	"github.com/LeFinal/masc-server/logging"
// 	mqtt "github.com/eclipse/paho.mqtt.golang"
// 	"go.uber.org/zap"
// 	"strings"
// )
//
// // shelliesShelly1RelayState is used for reading the relay-status for ShelliesShelly1.
// type shelliesShelly1RelayState string
//
// // shelliesShelly1RelayCommand is used for specifying the relay-command for ShelliesShelly1.
// type shelliesShelly1RelayCommand string
//
// type shelliesShelly1InputState int
//
// // Constants for ShelliesShelly1.
// const (
// 	shelliesShelly1RelayStateOn       shelliesShelly1RelayState   = "on"
// 	shelliesShelly1RelayStateOff      shelliesShelly1RelayState   = "off"
// 	shelliesShelly1RelayCommandOn     shelliesShelly1RelayCommand = "on"
// 	shelliesShelly1RelayCommandOff    shelliesShelly1RelayCommand = "off"
// 	shelliesShelly1RelayCommandToggle shelliesShelly1RelayCommand = "toggle"
// 	shelliesShelly1InputStateHigh     shelliesShelly1InputState   = 1
// 	shelliesShelly1InputStateLow      shelliesShelly1InputState   = 0
// )
//
// type shelliesShelly1State struct {
// 	relayState shelliesShelly1RelayState
// 	inputState shelliesShelly1InputState
// }
//
// type netShelliesShelly1 struct {
// 	// shellyMQTTID is the device id of the shelly. Taken from topic format:
// 	//  shellies/shelly1-<device-id>/...
// 	shellyMQTTID string
// 	// deviceID is the actually assigned device id from MASC.
// 	deviceID events.DeviceID
// 	// state is the known state of shelly.
// 	state shelliesShelly1State
// 	// isStateInitialized describes whether the state was set from the first fixture
// 	// state message. This is used in order to send an update with the initial state
// 	// even if the default values match the first applied fixture state.
// 	isStateInitialized bool
// 	// fixtureProviderActorID is the assigned actor id for
// 	// acting.RoleTypeFixtureProvider.
// 	fixtureProviderActorID events.ActorID
// 	// lightSwitchProviderActorID is the assigned actor id for
// 	// acting.RoleTypeLightSwitchProvider.
// 	lightSwitchProviderActorID events.ActorID
// 	// No need for synchronization as we handle message after message.
// }
//
// func newNetShelliesShelly1(topic string) (*netShelliesShelly1, error) {
// 	// Extract shelly device id.
// 	topicSegments := strings.Split(topic, "/")
// 	if len(topicSegments) < 2 {
// 		return nil, errors.NewInternalError("expected at least 2 segments in topic", errors.Details{"mqtt_topic": topic})
// 	}
// 	idSegments := strings.Split(topicSegments[1], "shelly1-")
// 	if len(idSegments) != 2 {
// 		return nil, errors.NewInternalError("expected 2 id segments in topic segment", errors.Details{
// 			"mqtt_topic":                  topic,
// 			"mqtt_topic_selected_segment": topicSegments[1],
// 			"mqtt_id_segments":            idSegments,
// 		})
// 	}
// 	return &netShelliesShelly1{
// 		shellyMQTTID: idSegments[1],
// 		state: shelliesShelly1State{
// 			relayState: shelliesShelly1RelayStateOff,
// 			inputState: shelliesShelly1InputStateLow,
// 		},
// 	}, nil
// }
//
// func (shelly *netShelliesShelly1) genMessageHello() events.MessageHello {
// 	return events.MessageHello{
// 		Roles: []events.Role{
// 			events.Role(acting.RoleTypeFixtureProvider),
// 			events.Role(acting.RoleTypeLightSwitchProvider),
// 		},
// 		SelfDescription: "shelly1",
// 	}
// }
//
// func (shelly *netShelliesShelly1) run(ctx context.Context, deviceID events.DeviceID, fromMQTT chan mqtt.Message, fromMASC chan events.EventContainer, publisher publisher) error {
// 	shelly.deviceID = deviceID
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case message := <-fromMQTT:
// 			if err := shelly.handleFromMQTT(ctx, message, publisher); err != nil {
// 				return errors.Wrap(err, "handle from mqtt", nil)
// 			}
// 		case message := <-fromMASC:
// 			if err := shelly.handleFromMASC(ctx, message, publisher); err != nil {
// 				return errors.Wrap(err, "handle from masc", nil)
// 			}
// 		}
// 	}
// }
//
// func (shelly *netShelliesShelly1) handleFromMQTT(ctx context.Context, message mqtt.Message, publisher publisher) error {
// 	switch {
// 	case strings.HasSuffix(message.Topic(), "/relay/0"):
// 		if err := shelly.handleRelayStateMessage(ctx, message, publisher); err != nil {
// 			return errors.Wrap(err, "handle relay state message", nil)
// 		}
// 	case strings.HasSuffix(message.Topic(), "/input/0"):
// 		if err := shelly.handleInputStateMessage(ctx, message, publisher); err != nil {
// 			return errors.Wrap(err, "handle input state message", nil)
// 		}
// 	}
// 	// Not found.
// 	return nil
// }
//
// func (shelly *netShelliesShelly1) handleRelayStateMessage(ctx context.Context, message mqtt.Message, publisher publisher) error {
// 	attempted := shelliesShelly1RelayState(message.Payload())
// 	if shelly.state.relayState == attempted {
// 		return nil
// 	}
// 	// Overwrite.
// 	if err := shelly.publishFixtureState(ctx, publisher); err != nil {
// 		return errors.Wrap(err, "overwrite fixture state", errors.Details{"attempted_relay_state": attempted})
// 	}
// 	return nil
// }
//
// func (shelly *netShelliesShelly1) handleInputStateMessage(ctx context.Context, message mqtt.Message, publisher publisher) error {
// 	if len(shelly.lightSwitchProviderActorID) == 0 {
// 		logging.MQTTLogger.Debug("not publishing light switch state due to no actor id being assigned",
// 			zap.Any("shelly_mqtt_id", shelly.shellyMQTTID))
// 		return nil
// 	}
// 	if err := publisher.publishMASC(ctx, events.MessageTypeLightSwitchHiLoState, shelly.lightSwitchProviderActorID,
// 		events.MessageLightSwitchHiLoState{
// 			IsHigh: string(message.Payload()) == "1",
// 		}); err != nil {
// 		return errors.Wrap(err, "publish masc", nil)
// 	}
// 	return nil
// }
//
// func (shelly *netShelliesShelly1) handleFromMASC(ctx context.Context, message events.EventContainer, publisher publisher) error {
// 	// A bit hacky, but we do not expect two messages with the same type for
// 	// different roles.
// 	switch message.MessageType {
// 	case events.MessageTypeYouAreIn:
// 		// Parse content.
// 		var messageContent events.MessageYouAreIn
// 		if err := json.Unmarshal(message.Content, &messageContent); err != nil {
// 			return errors.NewJSONError(err, "unmarshal you-are-in message content", false)
// 		}
// 		// Set assigned actor id.
// 		switch messageContent.Role {
// 		case events.Role(acting.RoleTypeFixtureProvider):
// 			shelly.fixtureProviderActorID = messageContent.ActorID
// 		case events.Role(acting.RoleTypeLightSwitchProvider):
// 			shelly.lightSwitchProviderActorID = messageContent.ActorID
// 		default:
// 			return errors.NewInternalError("assignment of unexpected role", errors.Details{
// 				"role": messageContent.Role,
// 			})
// 		}
// 		return nil
// 	case events.MessageTypeFixtureBasicState:
// 		// Parse content.
// 		var messageContent events.MessageFixtureBasicState
// 		if err := json.Unmarshal(message.Content, &messageContent); err != nil {
// 			return errors.NewJSONError(err, "unmarshal fixture state", false)
// 		}
// 		// Apply.
// 		if err := shelly.applyFixtureState(ctx, messageContent, publisher); err != nil {
// 			return errors.Wrap(err, "apply fixture state", errors.Details{"fixture_state": messageContent})
// 		}
// 	case events.MessageTypeGetFixtureOffers:
// 		if err := shelly.publishFixtureOffers(ctx, publisher); err != nil {
// 			return errors.Wrap(err, "publish fixture offers", nil)
// 		}
// 	case events.MessageTypeGetLightSwitchOffers:
// 		if err := shelly.publishLightSwitchOffers(ctx, publisher); err != nil {
// 			return errors.Wrap(err, "publish light switch offers", nil)
// 		}
// 	}
// 	// Unknown message.
// 	return nil
// }
//
// func (shelly *netShelliesShelly1) applyFixtureState(ctx context.Context, fixtureState events.MessageFixtureBasicState, publisher publisher) error {
// 	if shelly.isStateInitialized && genShelliesShelly1FixtureState(shelly.state) == fixtureState {
// 		return nil
// 	}
// 	if err := shelly.setRelayState(ctx, fixtureState.IsEnabled, publisher); err != nil {
// 		return errors.Wrap(err, "set relay state", errors.Details{"relay_enabled": fixtureState.IsEnabled})
// 	}
// 	return nil
// }
//
// // setRelayState sets the relay to the given enabled-state and publishes the update.
// func (shelly *netShelliesShelly1) setRelayState(ctx context.Context, isEnabled bool, publisher publisher) error {
// 	if isEnabled {
// 		shelly.state.relayState = shelliesShelly1RelayStateOn
// 	} else {
// 		shelly.state.relayState = shelliesShelly1RelayStateOff
// 	}
// 	err := publisher.publishMQTT(ctx, fmt.Sprintf("shellies/shelly1-%s/relay/0/command", shelly.shellyMQTTID),
// 		string(shelly.state.relayState))
// 	if err != nil {
// 		return errors.Wrap(err, "publish mqtt", nil)
// 	}
// 	return nil
// }
