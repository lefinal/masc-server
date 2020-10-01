// Common messages for gatekeeping. This includes logging in users or other control messages.

package messages

// The message types
const (
	MsgTypeHello   MessageType = "hello"
	MsgTypeWelcome MessageType = "welcome"
)

// HelloMessage is received when a device wants to login.
type HelloMessage struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
}

type WelcomeMessage struct {
	ServerName string `json:"server_name"`
}
