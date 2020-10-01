// Common messages for gatekeeping. This includes logging in users or other control messages.

package messages

// The message types
const (
	MsgTypeHello   MessageType = "hello"
	MsgTypeWelcome MessageType = "welcome"
)

// HelloMessage is received when a device wants to login.
type HelloMessage struct {
	MessageMeta `json:"meta"`
	Name        string   `json:"name"`
	Roles       []string `json:"roles"`
}

type WelcomeMessage struct {
	MessageMeta `json:"meta"`
	ServerName  string `json:"server_name"`
}
