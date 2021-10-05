package errors

import (
	"reflect"
	"testing"
)

func TestNewResourceNotFoundError(t *testing.T) {
	type args struct {
		message string
		details Details
	}
	tests := []struct {
		name string
		args args
		want Error
	}{
		{
			name: "without details",
			args: args{
				message: "hello world",
				details: nil,
			},
			want: Error{
				Code:    ErrNotFound,
				Kind:    KindResourceNotFound,
				Err:     nil,
				Message: "hello world",
				Details: nil,
			},
		},
		{
			name: "with details",
			args: args{
				message: "hello world",
				details: Details{"hello": "world"},
			},
			want: Error{
				Code:    ErrNotFound,
				Kind:    KindResourceNotFound,
				Err:     nil,
				Message: "hello world",
				Details: Details{"hello": "world"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err, ok := Cast(NewResourceNotFoundError(tt.args.message, tt.args.details)); !ok || !reflect.DeepEqual(err, tt.want) {
				t.Errorf("NewResourceNotFoundError() error = %v, ok = %v, want %v, ok = %v", err, ok, tt.want, true)
			}
		})
	}
}
