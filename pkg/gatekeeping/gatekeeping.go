// Common messages for gatekeeping. This includes logging in users or other control messages.

package gatekeeping

import (
	"masc-server/pkg/errors"
	"masc-server/pkg/networking"
)

// The message types
const (
	MsgTypeHello   networking.MessageType = "hello"
	MsgTypeWelcome networking.MessageType = "welcome"
)

// The error codes
const (
	ErrIdAlreadyTaken errors.ErrorCode = "gatekeeping.id-already-taken"
)

// HelloMessage is received when a device wants to login.
type HelloMessage struct {
	networking.MessageMeta `json:"meta"`
	Name                   string   `json:"name"`
	Roles                  []string `json:"roles"`
}

type WelcomeMessage struct {
	networking.MessageMeta `json:"meta"`
	ServerName             string `json:"server_name"`
}
