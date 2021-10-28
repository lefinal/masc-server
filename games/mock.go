package games

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/google/uuid"
	"sync"
)

type MockPlayerProvider struct {
	knownPlayers      map[messages.UserID]struct{}
	knownPlayersMutex sync.RWMutex
}

func NewMockPlayerProvider() *MockPlayerProvider {
	return &MockPlayerProvider{
		knownPlayers: make(map[messages.UserID]struct{}),
	}
}

func (p *MockPlayerProvider) SetPlayerKnown(id messages.UserID) {
	p.knownPlayersMutex.Lock()
	defer p.knownPlayersMutex.Unlock()
	p.knownPlayers[id] = struct{}{}
}

func (p *MockPlayerProvider) GetUserByID(id messages.UserID) (stores.User, error) {
	p.knownPlayersMutex.RLock()
	defer p.knownPlayersMutex.RUnlock()
	if _, ok := p.knownPlayers[id]; !ok {
		return stores.User{}, errors.NewResourceNotFoundError(fmt.Sprintf("unknown user %v", id), errors.Details{})
	}
	return stores.User{ID: id}, nil
}

func (p *MockPlayerProvider) CreateGuestUser() (stores.User, error) {
	newID := messages.UserID(uuid.New().String())
	p.SetPlayerKnown(newID)
	return p.GetUserByID(newID)
}
