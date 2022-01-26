package lightswitch

import (
	"context"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"sync"
)

// SwitchBaseApplier is used for applying actions from light switches with type
// messages.LightSwitchTypeHiLo.
type SwitchBaseApplier interface {
	ToggleFixtureEnabledByID(id messages.FixtureID) error
}

// lightSwitchHiLo represents
type lightSwitchHiLo struct {
	lightSwitchBase
	applier SwitchBaseApplier
	// isHigh states whether we are currently high or not.
	isHigh bool
}

// newLightSwitchHiLo creates a new lightSwitchHiLo.
func newLightSwitchHiLo(id messages.LightSwitchID, applier SwitchBaseApplier) LightSwitch {
	return &lightSwitchHiLo{
		lightSwitchBase: newLightSwitchBase(id),
		applier:         applier,
	}
}

func (ls *lightSwitchHiLo) Type() messages.LightSwitchType {
	return messages.LightSwitchTypeHiLo
}

func (ls *lightSwitchHiLo) Features() []messages.LightSwitchFeature {
	return make([]messages.LightSwitchFeature, 0)
}

func (ls *lightSwitchHiLo) run(ctx context.Context, actor acting.Actor) error {
	ls.setRunning(true)
	defer ls.setRunning(false)
	sub := acting.SubscribeMessageTypeLightSwitchHiLoState(actor)
	for {
		select {
		case m := <-sub.Receive:
			ls.handleMessage(m)
		case <-ctx.Done():
			return ctx.Err()
		case <-actor.Quit():
			return nil
		}
	}
}

// handleMessage handles the state update.
func (ls *lightSwitchHiLo) handleMessage(m messages.MessageLightSwitchHiLoState) {
	ls.m.Lock()
	defer ls.m.Unlock()
	// We want to toggle the enabled state of the fixture when we receive a HI
	// signal.
	if !ls.isHigh && m.IsHigh {
		// Toggle.
		var wg sync.WaitGroup
		wg.Add(len(ls.assignments))
		for _, assignment := range ls.assignments {
			go func(fixtureID messages.FixtureID) {
				defer wg.Done()
				err := ls.applier.ToggleFixtureEnabledByID(fixtureID)
				if err != nil {
					errors.Log(logging.LightSwitchLogger, errors.Wrap(err, "toggle fixture enabled by id",
						errors.Details{"fixture_id": fixtureID}))
				}
			}(assignment)
		}
		wg.Wait()
	}
	// Set state.
	ls.isHigh = m.IsHigh
}
