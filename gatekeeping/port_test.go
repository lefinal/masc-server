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
				Send:                 tt.fields.Send,
			}
			fmt.Println("now sending message")
			port.handleMessageToBeSent(tt.args.container)
			fmt.Println("message passed")
			assert.NotEmpty(t, port.client.(*MockClient).Sent, "message should be sent")
		})
	}
}

func TestGatePort_handleReceivedMessage(t *testing.T) {
	type args struct {
		messages []networking.InboundMessage
	}
	helloMessage := messages.HelloMessage{
		Name:        "Test client 1",
		Description: "A test client which does absolutely nothing",
		Roles:       []string{"hello"},
	}
	getScheduleMessageStr := messages.MarshalMessageFromMetaAndPayloadMust(messages.MessageMeta{
		Type:     string(messages.MsgTypeGetSchedule),
		DeviceId: uuid.UUID{},
	}, messages.GetScheduleMessage{})
	newMatchMessageStr := messages.MarshalMessageFromMetaAndPayloadMust(messages.MessageMeta{
		Type:     string(messages.MsgTypeNewMatch),
		DeviceId: uuid.New(),
	}, messages.NewMatchMessage{})

	tests := []struct {
		name                 string
		args                 args
		wantReceivedMessages int
		wantSentMessages     int
		wantSentErrMessages  int
	}{
		{
			name: "Send correct first message",
			args: args{
				messages: []networking.InboundMessage{{
					Message: messages.MarshalMessageFromMetaAndPayloadMust(messages.MessageMeta{
						Type:     string(messages.MsgTypeHello),
						DeviceId: uuid.UUID{},
					}, helloMessage),
				}},
			},
			wantReceivedMessages: 1,
			wantSentMessages:     0,
			wantSentErrMessages:  0,
		},
		{
			name: "Send first message with set uuid.",
			args: args{
				messages: []networking.InboundMessage{{
					Message: messages.MarshalMessageFromMetaAndPayloadMust(messages.MessageMeta{
						Type:     string(messages.MsgTypeHello),
						DeviceId: uuid.New(),
					}, helloMessage),
				}},
			},
			wantReceivedMessages: 1,
			wantSentMessages:     0,
			wantSentErrMessages:  0,
		},
		{
			name: "Send first message with not allowed type.",
			args: args{
				messages: []networking.InboundMessage{{
					Message: newMatchMessageStr,
				}},
			},
			wantReceivedMessages: 0,
			wantSentMessages:     1,
			wantSentErrMessages:  1,
		},
		{
			name: "Send first multiple not allowed messages.",
			args: args{
				messages: []networking.InboundMessage{{
					Message: newMatchMessageStr,
				}, {
					Message: getScheduleMessageStr,
				}, {
					Message: getScheduleMessageStr,
				}},
			},
			wantReceivedMessages: 0,
			wantSentMessages:     3,
			wantSentErrMessages:  3,
		},
		{
			name: "Send first multiple not allowed messages and then a correct one.",
			args: args{
				messages: []networking.InboundMessage{{
					Message: newMatchMessageStr,
				}, {
					Message: getScheduleMessageStr,
				}, {
					Message: messages.MarshalMessageFromMetaAndPayloadMust(messages.MessageMeta{
						Type:     string(messages.MsgTypeHello),
						DeviceId: uuid.UUID{},
					}, helloMessage),
				}},
			},
			wantReceivedMessages: 1,
			wantSentMessages:     2,
			wantSentErrMessages:  2,
		},
		{
			name: "Send first message with incorrect JSON",
			args: args{
				messages: []networking.InboundMessage{{
					Message: "{hello=world}",
				}},
			},
			wantReceivedMessages: 0,
			wantSentMessages:     1,
			wantSentErrMessages:  1,
		},
		{
			name: "Send first hello message with incorrect JSON payload",
			args: args{
				messages: []networking.InboundMessage{{
					Message: fmt.Sprintf("{\"meta\":{\"type\":\"%s\",\"device_id\":\"\"},\"payload\":{hello=world}}",
						messages.MsgTypeHello),
				}},
			},
			wantReceivedMessages: 0,
			wantSentMessages:     1,
			wantSentErrMessages:  1,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			employer := newMockEmployer()
			client := NewMockClient()
			port := newGatePort(client, employer)
			for _, message := range tt.args.messages {
				port.handleReceivedMessage(message)
			}
			assert.Eventuallyf(t, func() bool {
				correctReceivedMessages := len(employer.Inbox) == tt.wantReceivedMessages
				correctSentMessages := len(client.Sent) == tt.wantSentMessages
				correctSentErrorMessages := port.SentErrorMessages == tt.wantSentErrMessages
				return correctReceivedMessages && correctSentMessages && correctSentErrorMessages
			}, WaitTimeout, TickInterval, "expected %d message(s) to be received, %d message(s) to be sent and %d "+
				"sent error messages, but got %d received messages, %d sent messages and %d sent error messages.",
				tt.wantReceivedMessages, tt.wantSentMessages, tt.wantSentErrMessages,
				len(employer.Inbox), len(client.Sent), port.SentErrorMessages)
		})
	}
}
