package games

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/gobuffalo/nulls"
	"sync"
)

// CastingKey is the key for requested actors via casting.
type CastingKey string

// ActorRequest orders an acting.Actor for the given Role. Order is
// performed by calling roleAssigner.RequestActor.
type ActorRequest struct {
	// DisplayedName is the human-readable role name that is going to be used by the
	// client.
	DisplayedName string
	// Key is used in order to identify the request when retrieving winners.
	Key CastingKey
	// Role is the actual role the actor must satisfy.
	Role acting.Role
	// Min is an optional minimum count of winners.
	Min nulls.UInt32
	// Max is an optional maximum count of winners.
	Max nulls.UInt32
}

// castingState is used by casting in order to check if further role
// requesting, performing, etc. is allowed.
type castingState int

const (
	// castingStateReady is used when the casting is ready to accept orders
	// and start assignment.
	castingStateReady castingState = iota
	// castingStateCasting is used when the casting is currently ongoing.
	castingStateCasting
	// castingStateDone is used when the assigner has finished assigning or an
	// error occurred.
	castingStateDone
)

type casting struct {
	// agency is where actors are retrieved and hired from.
	agency acting.Agency
	// jury is the acting.Actor that performs the casting and determines winners.
	jury acting.Actor
	// requests holds all requests to be done.
	requests []*ActorRequest
	// requestsMutex is a lock for requests.
	requestsMutex sync.RWMutex
	// ordersRemaining is used for waiting until all requests where satisfied.
	ordersRemaining sync.WaitGroup
	// state is the state of the assigner.
	state castingState
	// stateMutex is a lock for state.
	stateMutex sync.RWMutex
	// winners are the hired actors from the casting.
	winners map[CastingKey][]acting.Actor
	// winnersMutex is a lock for winners.
	winnersMutex sync.RWMutex
}

// NewCasting creates a new role assigner that is used in order to hire
// actors for roles. The assigner is safe for concurrent access.
//
// Warning: Make sure that the agency provides a RoleTypeRoleAssigner that is
// available when performing assignments.
func NewCasting(agency acting.Agency) *casting {
	return &casting{
		agency:   agency,
		requests: make([]*ActorRequest, 0),
	}
}

// assureCastingRequestsAlterAllowed makes sure that changes to order of a
// casting are allowed. This is done by checking the passed state and
// creating appropriate errors.
func assureCastingRequestsAlterAllowed(state castingState) error {
	switch state {
	case castingStateCasting:
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindAlreadyPerformingCasting,
			Message: "already performing casting",
		}
	case castingStateDone:
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindCastingAlreadyDone,
			Message: "casting already done",
		}
	case castingStateReady:
		return nil
	default:
		// Unknown state.
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindUnknown,
			Message: fmt.Sprintf("unknown casting state: %d", state),
			Details: errors.Details{"state": state},
		}
	}
}

// assureValidActorRequest assures that the ActorRequest.Min and
// ActorRequest.Max values are set correctly as well as checking if a request
// was already added for the same ActorRequest.Key.
func (c *casting) assureValidActorRequest(request ActorRequest) error {
	defer c.requestsMutex.RUnlock()
	c.requestsMutex.RLock()
	// Check if request was already made.
	for _, actorRequest := range c.requests {
		if actorRequest.Key == request.Key {
			return errors.Error{
				Code:    errors.ErrInternal,
				Kind:    errors.KindInvalidCastingRequest,
				Message: fmt.Sprintf("request for casting key %v already exists", request.Key),
				Details: errors.Details{"castingKey": request.Key, "request": request},
			}
		}
	}
	// Check min and max values.
	if request.Max.Valid && request.Max.UInt32 < 1 {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindInvalidCastingRequest,
			Message: fmt.Sprintf("max value must not be less than 1 but was %d", request.Max.UInt32),
			Details: errors.Details{"castingKey": request.Key, "request": request, "max": request.Max.UInt32},
		}
	}
	if request.Min.Valid && request.Max.Valid && request.Min.UInt32 > request.Max.UInt32 {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindInvalidCastingRequest,
			Message: fmt.Sprintf("max value must not exceed min value %d but was %d", request.Min.UInt32, request.Max.UInt32),
			Details: errors.Details{"castingKey": request.Key, "request": request, "max": request.Max.UInt32},
		}
	}
	return nil
}

// AddRequest requests the assigner to hire an acting.Actor for the role in
// the given ActorRequest. Assignment is performed via PerformAndHire.
func (c *casting) AddRequest(request ActorRequest) error {
	defer c.stateMutex.RUnlock()
	// Check if request is allowed.
	c.stateMutex.RLock()
	err := assureCastingRequestsAlterAllowed(c.state)
	if err != nil {
		return err
	}
	// Assure valid request.
	err = c.assureValidActorRequest(request)
	if err != nil {
		return errors.Wrap(err, "assure valid actor request")
	}
	// Add order and wait for response.
	c.requestsMutex.Lock()
	c.requests = append(c.requests, &request)
	c.requestsMutex.Unlock()
	return nil
}

// PerformAndHire allows performing the actual casting and hiring winners.
func (c *casting) PerformAndHire(ctx context.Context, jury acting.Actor) error {
	// Check if allowed.
	c.stateMutex.Lock()
	err := assureCastingRequestsAlterAllowed(c.state)
	if err != nil {
		c.stateMutex.Unlock()
		return err
	}
	// Change state.
	c.state = castingStateCasting
	c.stateMutex.Unlock()
	// Hello jury!
	c.jury = jury
	// Build orders.
	c.requestsMutex.RLock()
	orders := make([]messages.RoleAssignmentOrder, len(c.requests))
	for i, request := range c.requests {
		orders[i] = messages.RoleAssignmentOrder{
			DisplayedName: request.DisplayedName,
			Key:           string(request.Key),
			Role:          messages.Role(request.Role),
			Min:           request.Min,
			Max:           request.Max,
		}
	}
	c.requestsMutex.RUnlock()
	// Try to assign until valid or context done.
	for {
		// Subscribe to assignments. I know, that if the context is cancelled, this is
		// bs, but nah.
		newsletterAssignments := acting.SubscribeMessageTypeRoleAssignments(c.jury)
		// Send role assignment request to assigner.
		err = c.jury.Send(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypeRequestRoleAssignments,
			Content: messages.MessageRequestRoleAssignments{
				Orders: nil,
			},
		})
		if err != nil {
			return errors.Wrap(err, "request role assignments")
		}
		// Wait for reply.
		select {
		case <-ctx.Done():
			acting.UnsubscribeOrLogError(newsletterAssignments.Newsletter)
			return errors.NewContextAbortedError("perform casting")
		case messageAssignments, ok := <-newsletterAssignments.Receive:
			if !ok {
				return errors.Error{
					Code:    errors.ErrInternal,
					Kind:    errors.KindCastingAborted,
					Message: "no assignments received",
				}
			}
			acting.UnsubscribeOrLogError(newsletterAssignments.Newsletter)
			// Handle assignments by first checking if the request was fulfilled correctly.
			if badRequestErr := c.setWinners(messageAssignments); badRequestErr != nil {
				notifyErr := c.jury.Send(acting.ActorErrorMessageFromError(errors.Wrap(badRequestErr,
					"set winners from assignment message")))
				if notifyErr != nil {
					return errors.Wrap(err, "notify jury for invalid role assignments")
				}
				continue
			}
			// Assure valid assignment.
			if badRequestErr := c.assureWinnersSatisfyRequests(); badRequestErr != nil {
				notifyErr := c.jury.Send(acting.ActorErrorMessageFromError(errors.Wrap(badRequestErr,
					"assure winners satisfy requests")))
				if notifyErr != nil {
					return errors.Wrap(err, "notify jury for invalid role assignments")
				}
				continue
			}
			// Hire winners.
			badRequestErr, internalErr := c.hireWinners()
			if internalErr != nil {
				return errors.Wrap(internalErr, "hire winners")
			}
			if badRequestErr != nil {
				notifyErr := c.jury.Send(acting.ActorErrorMessageFromError(errors.Wrap(badRequestErr,
					"hire winners from assignment message")))
				if notifyErr != nil {
					return errors.Wrap(err, "notify jury for invalid role assignments")
				}
				continue
			}
		}
		// Assignment was successful.
		break
	}
	// All done.
	c.stateMutex.Lock()
	c.state = castingStateDone
	c.stateMutex.Unlock()
	return nil
}

// hireWinners hires all casting.winners. If a winner cannot be hired, all
// already hired ones will get fired. If a winners cannot be hired and the
// already hired winners get fired successfully, an errors.ErrBadRequest is
// returned as first result. If the second case is not true, an errors.ErrInternal is returned as second result.
func (c *casting) hireWinners() (error, error) {
	defer c.winnersMutex.Unlock()
	c.winnersMutex.Lock()
	hired := make([]acting.Actor, 0)
	for castingKey, actors := range c.winners {
		for _, actor := range actors {
			if hireError := actor.Hire(); hireError != nil {
				// Fire already hired actors.
				if fireAllErr := acting.FireAllActors(hired); fireAllErr != nil {
					return nil, errors.Wrap(fireAllErr, "fire all actors because of failed hire after role assignments")
				} else {
					return errors.Error{
						Code:    errors.ErrBadRequest,
						Kind:    errors.KindInvalidRoleAssignments,
						Message: fmt.Sprintf("hiring assigned actor for casting key %v failed: %s", castingKey, hireError.Error()),
						Details: errors.Details{
							"alreadyHired": len(hired),
							"castingKey":   castingKey,
							"hireError":    hireError,
						},
					}, nil
				}
			}
			hired = append(hired, actor)
		}
	}
	return nil, nil
}

// assureWinnersSatisfyRequests assures that all requests were satisfied by
// checking the ActorRequest.Min and ActorRequest.Max values. If this is not the
// case, an errors.ErrBadRequest is returned.
func (c *casting) assureWinnersSatisfyRequests() error {
	defer c.requestsMutex.RUnlock()
	c.requestsMutex.RLock()
	for _, request := range c.requests {
		// Check if minimum count is not satisfied.
		assignmentsCount := uint32(len(c.winners[request.Key]))
		if request.Min.Valid && assignmentsCount < request.Min.UInt32 {
			return errors.Error{
				Code: errors.ErrBadRequest,
				Kind: errors.KindInvalidRoleAssignments,
				Message: fmt.Sprintf("assignments count %d for request %v does match minimum of %d",
					assignmentsCount, request.Key, request.Min.UInt32),
				Details: errors.Details{"actual": assignmentsCount, "expected": request.Min.UInt32},
			}
		}
		// Check if maximum count is not satisfied.
		if request.Max.Valid && assignmentsCount > request.Max.UInt32 {
			return errors.Error{
				Code: errors.ErrBadRequest,
				Kind: errors.KindInvalidRoleAssignments,
				Message: fmt.Sprintf("assignments count %d for request %v does match maximum of %d",
					assignmentsCount, request.Key, request.Max.UInt32),
				Details: errors.Details{"actual": assignmentsCount, "expected": request.Max.UInt32},
			}
		}
	}
	return nil
}

// setWinnersFromAssignmentsMessages sets casting.winners based on the given
// messages.MessageRoleAssignments. It also initializes an empty list for each
// casting.requests which allows generating errors for winner retrieval with
// unknown casting keys. Returns an errors.ErrBadRequest error when the assigned
// Actor was not found with the given id.
func (c *casting) setWinners(messageAssignments messages.MessageRoleAssignments) error {
	defer c.winnersMutex.Unlock()
	defer c.requestsMutex.Unlock()
	c.winnersMutex.Lock()
	c.requestsMutex.Lock()
	c.winners = make(map[CastingKey][]acting.Actor)
	// Create empty lists for requests.
	for _, request := range c.requests {
		c.winners[request.Key] = make([]acting.Actor, 0)
	}
	// Add winners.
	for _, assignment := range messageAssignments.Assignments {
		actor, found := c.agency.ActorByID(assignment.ActorID)
		if !found {
			return errors.Error{
				Code:    errors.ErrBadRequest,
				Kind:    errors.KindUnknownActor,
				Message: fmt.Sprintf("assigned actor %v not found", assignment.ActorID),
				Details: errors.Details{"actorID": assignment.ActorID},
			}
		}
		c.winners[CastingKey(assignment.Key)] = append(c.winners[CastingKey(assignment.Key)], actor)
	}
	return nil
}

// GetWinner serves as a wrapper for GetWinners that assures exactly 1 winner.
func (c *casting) GetWinner(key CastingKey) (acting.Actor, error) {
	winners, err := c.GetWinners(key)
	if err != nil {
		return nil, err
	}
	if len(winners) != 1 {
		return nil, errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindCountDoesNotMatchExpected,
			Message: fmt.Sprintf("winner count %d does not match expected %d", len(winners), 1),
		}
	}
	return winners[0], nil
}

// GetWinners returns the winners for the given CastingKey.
func (c *casting) GetWinners(key CastingKey) ([]acting.Actor, error) {
	defer c.stateMutex.Unlock()
	c.stateMutex.Lock()
	// Check if retrieval allowed.
	if c.state != castingStateDone {
		return nil, errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindCastingNotDone,
			Message: fmt.Sprintf("casting not done yet"),
			Details: errors.Details{"state": c.state},
		}
	}
	// Retrieve.
	c.winnersMutex.RLock()
	winners, found := c.winners[key]
	c.winnersMutex.RUnlock()
	if !found {
		return nil, errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindUnknownCastingKey,
			Message: fmt.Sprintf("unknown casting key: %v", key),
			Details: errors.Details{"castingKey": key},
		}
	}
	return winners, nil
}
