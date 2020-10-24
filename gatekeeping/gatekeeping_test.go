package gatekeeping

import (
	"github.com/LeFinal/masc-server/config"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/networking"
	"github.com/stretchr/testify/assert"
	"testing"
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

type MockEmployer struct {
	NewGatePorts    chan *GatePort
	Inbox           chan messages.MessageContainer
	ClosedGatePorts chan *GatePort
}

func newMockEmployer() *MockEmployer {
	return &MockEmployer{
		NewGatePorts:    make(chan *GatePort, 32),
		Inbox:           make(chan messages.MessageContainer, 32),
		ClosedGatePorts: make(chan *GatePort, 32),
	}
}

func (m *MockEmployer) AcceptNewGatePort(port *GatePort) {
	m.NewGatePorts <- port
}

func (m *MockEmployer) HandleClosedGatePort(port *GatePort) {
	panic("implement me")
}

func (m *MockEmployer) AcceptNewMessage(container messages.MessageContainer) {
	m.Inbox <- container
}

func TestGatekeeper_handleNewClient(t *testing.T) {
	type fields struct {
		employer *MockEmployer
	}
	type args struct {
		c networking.Client
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "New client",
			fields: fields{
				employer: newMockEmployer(),
			},
			args: args{
				c: NewMockClient(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gk := NewGateKeeper(config.NetworkConfig{}, tt.fields.employer)
			gk.handleNewClient(tt.args.c)
			assert.NotEmpty(t, tt.fields.employer.NewGatePorts,
				"employer should have received a new gate port")
		})
	}
}
