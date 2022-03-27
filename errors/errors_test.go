package errors

import (
	"errors"
	"reflect"
	"testing"
)

func TestCast(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name   string
		args   args
		want   Error
		wantOK bool
	}{
		{
			name: "with rich error",
			args: args{
				err: Error{
					Code:    ErrBadRequest,
					Err:     nil,
					Message: "this was a bad request",
				},
			},
			want: Error{
				Code:    ErrBadRequest,
				Err:     nil,
				Message: "this was a bad request",
			},
			wantOK: true,
		},
		{
			name: "with rich error and original error",
			args: args{
				err: Error{
					Code:    ErrBadRequest,
					Err:     errors.New("i am an error"),
					Message: "this was a bad request",
				},
			},
			want: Error{
				Code:    ErrBadRequest,
				Err:     errors.New("i am an error"),
				Message: "this was a bad request",
			},
			wantOK: true,
		},
		{
			name: "with nil error",
			args: args{
				err: nil,
			},
			want: Error{
				Code:    ErrUnexpected,
				Err:     nil,
				Message: "unknown operation",
				Details: make(Details),
			},
			wantOK: false,
		},
		{
			name: "with simple error",
			args: args{
				err: errors.New("i am an error"),
			},
			want: Error{
				Code:    ErrUnexpected,
				Err:     errors.New("i am an error"),
				Message: "unknown operation",
				Details: make(Details),
			},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := Cast(tt.args.err); !reflect.DeepEqual(got, tt.want) || ok != tt.wantOK {
				t.Errorf("Cast() = %v, %v, want %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	type fields struct {
		Code    Code
		Err     error
		Message string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "example 0",
			fields: fields{
				Code:    ErrBadRequest,
				Err:     errors.New("hello world"),
				Message: "unknown operation",
			},
			want: "unknown operation: hello world",
		},
		{
			name: "example 1",
			fields: fields{
				Code:    ErrBadRequest,
				Err:     errors.New("hello world"),
				Message: "known operation",
			},
			want: "known operation: hello world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Error{
				Code:    tt.fields.Code,
				Err:     tt.fields.Err,
				Message: tt.fields.Message,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromErr(t *testing.T) {
	type args struct {
		message string
		code    Code
		err     error
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "example 0",
			args: args{
				message: "i am the message",
				code:    ErrProtocolViolation,
				err:     errors.New("i am the error"),
			},
			want: errors.New("i am the message: i am the error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := FromErr(tt.args.message, tt.args.code, tt.args.err, nil); err == nil || err.Error() != tt.want.Error() {
				t.Errorf("FromErr() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	type args struct {
		message string
		err     error
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "with rich error",
			args: args{
				message: "i am the wrapper",
				err: &Error{
					Code:    ErrNotFound,
					Err:     errors.New("i am the error"),
					Message: "i am the original operation",
				},
			},
			want: errors.New("i am the wrapper: i am the original operation: i am the error"),
		},
		{
			name: "with simple error",
			args: args{
				message: "i am the wrapper",
				err:     errors.New("i am the error"),
			},
			want: errors.New("i am the wrapper: i am the error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Wrap(tt.args.err, tt.args.message, nil); err == nil || err.Error() != tt.want.Error() {
				t.Errorf("Wrap() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBlameUser(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not found",
			args: args{
				err: Error{Code: ErrNotFound},
			},
			want: true,
		},
		{
			name: "bad request",
			args: args{
				err: Error{Code: ErrBadRequest},
			},
			want: true,
		},
		{
			name: "protocol violation",
			args: args{
				err: Error{Code: ErrProtocolViolation},
			},
			want: true,
		},
		{
			name: "internal",
			args: args{
				err: Error{Code: ErrInternal},
			},
			want: false,
		},
		{
			name: "communication",
			args: args{
				err: Error{Code: ErrCommunication},
			},
			want: false,
		},
		{
			name: "unexpected",
			args: args{
				err: errors.New("unknown error"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BlameUser(tt.args.err); got != tt.want {
				t.Errorf("BlameUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
