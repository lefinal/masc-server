package util

import (
	"encoding/json"
	"github.com/LeFinal/masc-server/errors"
)

// EncodeAsJSON serves as a wrapper for the standard encoding and error stuff.
func EncodeAsJSON(content interface{}) (json.RawMessage, error) {
	j, err := json.Marshal(content)
	if err != nil {
		return json.RawMessage{}, errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindEncodeJSON,
			Err:     err,
			Details: errors.Details{"content": content},
		}
	}
	return j, nil
}

// DecodeAsJSON serves as a wrapper for the standard decoding and error stuff.
func DecodeAsJSON(data json.RawMessage, target interface{}) error {
	err := json.Unmarshal(data, target)
	if err != nil {
		return errors.Error{
			Code:    errors.ErrBadRequest,
			Kind:    errors.KindDecodeJSON,
			Err:     err,
			Details: errors.Details{"content": data},
		}
	}
	return nil
}
