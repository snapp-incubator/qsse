package internal

import (
	"encoding/json"
	"errors"
)

type Error struct {
	Err  error `json:"err,omitempty"`
	Code int   `json:"code,omitempty"`
}

const ErrorTopic = "error"

var ErrNotAuthorized = errors.New("not authorized")

const (
	CodeNotAuthorized = iota + 1
	CodeTopicNotAvailable
)

func ErrTopicNotAvailable(topic string) Error {
	return Error{Err: errors.New("topic not available: " + topic), Code: CodeTopicNotAvailable}
}

func UnmarshalError(bytes []byte) Error {
	var err Error
	json.Unmarshal(bytes, &err)
	return err
}
