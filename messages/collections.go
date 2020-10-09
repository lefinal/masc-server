package messages

import "strings"

var (
	// AllowedMessagesLoggedOut contains the allowed messages for when a device is not logged in yet.
	AllowedMessagesLoggedOut = []MessageType{MsgTypeHello}
)

// AllowedMessageCollection holds a set of allowed message types and provides methods such as IsAllowed to check type.
type AllowedMessageCollection struct {
	allowedMessages map[MessageType]bool
}

// NewAllowedMessageCollection creates a new message collection from given allowed messages.
func NewAllowedMessageCollection(messages []MessageType) *AllowedMessageCollection {
	res := &AllowedMessageCollection{}
	res.SetAllowedMessages(messages)
	return res
}

// SetAllowedMessages uses a message type slice and saves them to an internal collection.
func (collection *AllowedMessageCollection) SetAllowedMessages(messages []MessageType) {
	collection.allowedMessages = make(map[MessageType]bool)
	for _, message := range messages {
		collection.allowedMessages[message] = true
	}
}

// IsAllowed checks whether a given message type is allowed.
func (collection *AllowedMessageCollection) IsAllowed(msgType MessageType) bool {
	if _, exists := collection.allowedMessages[msgType]; !exists {
		return false
	}
	return true
}

func (collection *AllowedMessageCollection) String() string {
	var allowedMessagesSlice = make([]string, len(collection.allowedMessages))
	for messageType, _ := range collection.allowedMessages {
		allowedMessagesSlice = append(allowedMessagesSlice, messageType.String())
	}
	return strings.Join(allowedMessagesSlice, ", ")
}
