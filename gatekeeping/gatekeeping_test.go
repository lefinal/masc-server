package gatekeeping

import (
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/networking"
)

type MockGatePort struct {
	Receive chan messages.MessageContainer
	Sent    chan messages.MessageContainer
}

func NewMockGatePort() *MockGatePort {
	return &MockGatePort{
		Receive: make(chan messages.MessageContainer, 256),
		Sent:    make(chan messages.MessageContainer, 256),
	}
}

type MockClient struct {
	Receive chan networking.InboundMessage
	Sent    chan networking.OutboundMessage
}

func NewMockClient() *MockClient {
	return &MockClient{
		Receive: make(chan networking.InboundMessage, 256),
		Sent:    make(chan networking.OutboundMessage, 256),
	}
}

func (m *MockClient) SendMessage(msg networking.OutboundMessage) {
	m.Sent <- msg
}

func (m *MockClient) Inbox() chan networking.InboundMessage {
	return m.Receive
}
