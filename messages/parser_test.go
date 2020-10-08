package messages

import (
	"encoding/json"
	"github.com/LeFinal/masc-server/errors"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func convertToRawMessage(i interface{}) json.RawMessage {
	msg, _ := json.Marshal(i)
	return msg
}

func TestParseMessage(t *testing.T) {
	helloMessageMeta := MessageMeta{
		Type:     "hello",
		DeviceId: uuid.UUID{},
	}
	helloMessagePayload := HelloMessage{
		Name:        "Test device",
		Description: "A test device that is completely useless",
		Roles:       []string{"game-master"},
	}
	type args struct {
		msg []byte
	}
	tests := []struct {
		name  string
		args  args
		want  MessageMeta
		want1 interface{}
		want2 *errors.MascError
	}{
		{
			name: "Hello message",
			args: args{
				msg: MarshalMessageMust(GeneralMessage{
					MessageMeta: helloMessageMeta,
					Payload:     convertToRawMessage(helloMessagePayload),
				}),
			},
			want:  helloMessageMeta,
			want1: &helloMessagePayload,
			want2: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := ParseMessage(tt.args.msg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseMessage() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ParseMessage() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("ParseMessage() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}
