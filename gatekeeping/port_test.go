package gatekeeping

import (
	"fmt"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/networking"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	WaitTimeout  = time.Second
	TickInterval = time.Millisecond
)

func TestGatePort_handleMessageToBeSent(t *testing.T) {
	type fields struct {
		logger               logging.Logger
		AllowedMessages      *messages.AllowedMessageCollection
		DeviceId             uuid.UUID
		client               networking.Client
		receivedFirstMessage bool
		Receive              chan messages.MessageContainer
		Send                 chan messages.MessageContainer
	}
	type args struct {
		container messages.MessageContainer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Welcome message",
			fields: fields{
				logger:               logging.NewLogger("test"),
				AllowedMessages:      messages.NewAllowedMessageCollection(messages.AllowedMessagesLoggedOut),
				DeviceId:             uuid.UUID{},
				client:               NewMockClient(),
				receivedFirstMessage: false,
				Receive:              make(chan messages.MessageContainer, 256),
				Send:                 make(chan messages.MessageContainer, 256),
			},
			args: args{
				container: messages.MessageContainer{
					MessageType: "welcome",
					Payload:     `{"server_name":"test-server"}`,
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &GatePort{
				logger:               tt.fields.logger,
				AllowedMessages:      tt.fields.AllowedMessages,
				DeviceId:             tt.fields.DeviceId,
				client:               tt.fields.client,
				receivedFirstMessage: tt.fields.receivedFirstMessage,
				Receive:              tt.fields.Receive,
				Send:                 tt.fields.Send,
			}
			fmt.Println("now sending message")
			port.handleMessageToBeSent(tt.args.container)
			fmt.Println("message passed")
			assert.NotEmpty(t, port.client.(*MockClient).Sent, "message should be sent")
		})
	}
}
