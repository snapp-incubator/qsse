package internal

import (
	"encoding/json"
	"errors"
)

type Error struct {
	Code int            `json:"code,omitempty"`
	Data map[string]any `json:"data,omitempty"`
}

const ErrorTopic = "error"

var ErrNotAuthorized = errors.New("not authorized")

const (
	CodeNotAuthorized = iota + 1
	CodeTopicNotAvailable
)

func NewErr(code int, data map[string]any) *Error {
	return &Error{
		Code: code,
		Data: data,
	}
}

func UnmarshalError(bytes []byte) Error {
	var e Error

	if err := json.Unmarshal(bytes, &e); err != nil {
		checkError(err)
	}

	return e
}
