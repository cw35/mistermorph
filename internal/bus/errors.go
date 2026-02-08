package bus

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeBusClosed           ErrorCode = "BUS_CLOSED"
	CodeNoSubscriber        ErrorCode = "NO_SUBSCRIBER"
	CodeTopicAlreadyHandled ErrorCode = "TOPIC_ALREADY_HANDLED"
	CodeTopicFrozen         ErrorCode = "TOPIC_FROZEN"
	CodeInvalidMessage      ErrorCode = "INVALID_MESSAGE"
	CodeInvalidTopic        ErrorCode = "INVALID_TOPIC"
)

type BusError struct {
	Code ErrorCode
	Err  error
}

func (e *BusError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Err.Error())
}

func (e *BusError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapError(code ErrorCode, err error) error {
	if err == nil {
		return nil
	}
	return &BusError{Code: code, Err: err}
}

func ErrorCodeOf(err error) ErrorCode {
	var busErr *BusError
	if errors.As(err, &busErr) {
		return busErr.Code
	}
	return ""
}
