package games

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"sync"
)

// PlayerManagementTeamKey is a key for a team used in PlayerManagement.
type PlayerManagementTeamKey string

// PlayerManagementUpdate is an update that is published from PlayerManagement
// when a player joins or leaves.
type PlayerManagementUpdate struct {
	// User is the id of the user that joint or left.
	User messages.UserID
	// Team is the team the User joined or left.
	Team PlayerManagementTeamKey
	// HasJoined is true when the user joined. Otherwise, he left and HasJoined is
	// false.
	HasJoined bool
}

// PlayerManagement is a wrapper for a map with messages.UserID and
// PlayerManagementTeamKey that allows concurrent access. It also allows
// receiving updates for player joins and leaves.
type PlayerManagement struct {
	// active holds all active players along with their assigned team.
	active map[messages.UserID]PlayerManagementTeamKey
	// m locks active.
	m sync.RWMutex
	// updates is an optional update channel that sends a PlayerManagementUpdate
	// when a player joins or leaves.
	updates chan PlayerManagementUpdate
}

// PlayerManagementPlayer is used for retrieving active players with
// PlayerManagement.ActivePlayers.
type PlayerManagementPlayer struct {
	// Team is the team the player belongs to.
	Team PlayerManagementTeamKey
	// User is the associated user id.
	User messages.UserID
}

// NewPlayerManagement creates a new PlayerManagement that is ready to use. The
// passed update channel is optional. It receives when a player joins or leaves.
func NewPlayerManagement(updates chan PlayerManagementUpdate) *PlayerManagement {
	return &PlayerManagement{
		active:  make(map[messages.UserID]PlayerManagementTeamKey),
		updates: updates,
	}
}

// IsActive checks if the user with the given id is active.
func (pm *PlayerManagement) IsActive(id messages.UserID) bool {
	pm.m.RLock()
	defer pm.m.RUnlock()
	_, ok := pm.active[id]
	return ok
}

// AddPlayer adds the given id to the active ones for the given team. If the
// player is already active, false is returned and the player is not added.
func (pm *PlayerManagement) AddPlayer(id messages.UserID, team PlayerManagementTeamKey) bool {
	pm.m.Lock()
	defer pm.m.Unlock()
	if _, ok := pm.active[id]; ok {
		return false
	}
	pm.active[id] = team
	if pm.updates != nil {
		pm.updates <- PlayerManagementUpdate{
			User:      id,
			Team:      team,
			HasJoined: true,
		}
	}
	return true
}

// RemovePlayer removes the given id from the active ones. If the player is
// unknown, false is returned.
func (pm *PlayerManagement) RemovePlayer(id messages.UserID) bool {
	pm.m.Lock()
	defer pm.m.Unlock()
	team, ok := pm.active[id]
	if !ok {
		return false
	}
	delete(pm.active, id)
	if pm.updates != nil {
		pm.updates <- PlayerManagementUpdate{
			User:      id,
			Team:      team,
			HasJoined: false,
		}
	}
	return true
}

// PlayersInTeam counts the players in the team with the given key.
func (pm *PlayerManagement) PlayersInTeam(team PlayerManagementTeamKey) []PlayerManagementPlayer {
	pm.m.RLock()
	defer pm.m.RUnlock()
	players := make([]PlayerManagementPlayer, 0)
	for user, key := range pm.active {
		if key == team {
			players = append(players, PlayerManagementPlayer{
				Team: team,
				User: user,
			})
		}
	}
	return players
}

// ActivePlayers returns a list of all active players.
func (pm *PlayerManagement) ActivePlayers() []PlayerManagementPlayer {
	players := make([]PlayerManagementPlayer, 0, len(pm.active))
	for user, team := range pm.active {
		players = append(players, PlayerManagementPlayer{
			Team: team,
			User: user,
		})
	}
	return players
}

// PlayerProvider provides the methods needed in order to handle player joins
// and guest requests. Player quits are NOT handled.
type PlayerProvider interface {
	// GetUserByID retrieves the stores.User with the given user id.
	GetUserByID(id messages.UserID) (stores.User, error)
	// CreateGuestUser creates a guest account that can be used. This account will
	// be deleted after a while.
	CreateGuestUser() (stores.User, error)
}

// PlayerJoinOffice handles player joining. Set the fields, start it with Open
// and close it with Close. Beware that the office does not check whether we are
// already running or finished. Remember to set the JoinedPlayers map as this
// can be used for multiple offices. Note, that players can leave from all
type PlayerJoinOffice struct {
	// Team is the key of the team that is used in order to distinguish this office
	// from others when using shared PlayerManagement.
	Team PlayerManagementTeamKey
	// Logger for non-critical errors.
	Logger *logrus.Logger
	// PlayerProvider is a store interface for retrieving users and creating guest
	// ones.
	PlayerProvider PlayerProvider
	// stopPlayerJoinHandlers is called in Close in order to stop the player join
	// handlers.
	stopPlayerJoinHandlers context.CancelFunc
	// playerJoinHandlers holds all handlers that accept player joins.
	playerJoinHandlers *errgroup.Group
	// PlayerManagement holds all joined players. This is used in order to send
	// errors when a player tries to join multiple times. It also allows concurrent
	// access for usage with multiple offices.
	PlayerManagement *PlayerManagement
}

// Open opens the office. This sends a messages.MessageTypePlayerJoinOpen to all
// actors. It then accepts messages.MessageTypePlayerJoin and
// messages.MessageTypePlayerLeave until Close is called.
func (office *PlayerJoinOffice) Open(ctx context.Context, actors ...acting.Actor) {
	playerJoin, stopPlayerJoin := context.WithCancel(ctx)
	office.stopPlayerJoinHandlers = stopPlayerJoin
	office.playerJoinHandlers, _ = errgroup.WithContext(playerJoin)
	for _, actor := range actors {
		// Start handler.
		office.playerJoinHandlers.Go(office.handlePlayerJoins(playerJoin, actor))
	}
}

// handlePlayerJoins handles messages from the given actor with
// messages.MessageTypePlayerJoin and messages.MessageTypePlayerLeave.
func (office *PlayerJoinOffice) handlePlayerJoins(ctx context.Context, actor acting.Actor) func() error {
	return func() error {
		// Subscribe to player joins and lefts.
		playerJoinNewsletter := acting.SubscribeMessageTypePlayerJoin(actor)
		playerLeftNewsletter := acting.SubscribeMessageTypePlayerLeave(actor)
		// Notify player join open.
		err := actor.Send(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoinOpen,
			Content:     messages.MessagePlayerJoinOpen{Team: string(office.Team)},
		})
		if err != nil {
			return errors.Wrap(err, "notify player join open")
		}
		// Accept player joins until closed.
	handlePlayerJoins:
		for {
			select {
			case <-ctx.Done():
				break handlePlayerJoins
			case m := <-playerJoinNewsletter.Receive:
				var user stores.User
				if m.RequestGuest {
					// Create guest user.
					user, err = office.PlayerProvider.CreateGuestUser()
					if err != nil {
						return errors.Wrap(err, "request guest user")
					}
				} else {
					// Retrieve user from provider.
					user, err = office.PlayerProvider.GetUserByID(m.User)
					if err != nil {
						if errors.BlameUser(err) {
							if blameErr := actor.Send(acting.ActorErrorMessageFromError(err)); blameErr != nil {
								return errors.Wrap(blameErr, "notify actor for error after user retrieval")
							}
						} else {
							return errors.Wrap(err, "retrieve user")
						}
						continue
					}
				}
				// We check here if the user already joined. Of course, we could avoid the store
				// retrieval but this allows better code readability. If the user has not
				// already joined, he is added.
				if !office.PlayerManagement.AddPlayer(user.ID, office.Team) {
					// Already joined.
					if blameErr := actor.Send(acting.ActorErrorMessageFromError(errors.Error{
						Code:    errors.ErrBadRequest,
						Kind:    errors.KindPlayerAlreadyJoined,
						Message: fmt.Sprintf("player already joined match"),
						Details: errors.Details{"player": user.ID},
					})); blameErr != nil {
						return errors.Wrap(blameErr, "notify actor that player already joined")
					}
				}
			case m := <-playerLeftNewsletter.Receive:
				if !office.PlayerManagement.RemovePlayer(m.Player) {
					// Player not found in active ones.
					if blameErr := actor.Send(acting.ActorErrorMessageFromError(errors.Error{
						Code:    errors.ErrBadRequest,
						Kind:    errors.KindPlayerNotJoined,
						Message: fmt.Sprintf("player did not join the match"),
						Details: errors.Details{"player": m.Player},
					})); blameErr != nil {
						return errors.Wrap(blameErr, "notify actor that player did not join")
					}
				}
			}
		}
		// Notify player join closed.
		err = actor.Send(acting.ActorOutgoingMessage{MessageType: messages.MessageTypePlayerJoinClosed})
		if err != nil {
			return errors.Wrap(err, "notify player join closed")
		}
		acting.UnsubscribeOrLogError(playerJoinNewsletter.Newsletter)
		acting.UnsubscribeOrLogError(playerLeftNewsletter.Newsletter)
		return nil
	}
}

// Close closes the office. This sends a messages.MessageTypePlayerJoinClosed to
// all actors and stops accepting messages.MessageTypePlayerJoin.
func (office *PlayerJoinOffice) Close() error {
	office.stopPlayerJoinHandlers()
	return office.playerJoinHandlers.Wait()
}

// BroadcastPlayerManagementUpdate broadcasts the given PlayerManagementUpdate
// to all recipients.
func BroadcastPlayerManagementUpdate(update PlayerManagementUpdate, playerProvider PlayerProvider, recipients ...acting.Actor) error {
	// Build the message to broadcast.
	var broadcastMessage acting.ActorOutgoingMessage
	if update.HasJoined {
		// Request player details from provider.
		user, err := playerProvider.GetUserByID(update.User)
		if err != nil {
			// This is indeed a problem but not critical. We log the error and simply set
			// the id.
			return errors.Wrap(err, "get user details from provider")
		}
		broadcastMessage = acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoined,
			Content:     messages.MessagePlayerJoined{PlayerDetails: user.Message()},
		}
	} else {
		broadcastMessage = acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerLeft,
			Content:     messages.MessagePlayerLeft{Player: update.User},
		}
	}
	// Send to all participating actors.
	for _, actor := range recipients {
		err := actor.Send(broadcastMessage)
		if err != nil {
			return errors.Wrap(err, "broadcast player update")
		}
	}
	return nil
}

// BroadcastReadyStateUpdate broadcasts the given ReadyStateUpdate to all
// recipients.
func BroadcastReadyStateUpdate(update ReadyStateUpdate, recipients ...acting.Actor) error {
	// Build the message.
	broadcastMessage := acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeReadyStateUpdate,
		Content:     update.Message(),
	}
	// Broadcast to all participating actors.
	for _, actor := range recipients {
		err := actor.Send(broadcastMessage)
		if err != nil {
			return errors.Wrap(err, "broadcast ready-state update")
		}
	}
	return nil
}
