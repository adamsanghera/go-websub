package sql

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrMalformedHub is returned when a hub fails to validate
	ErrMalformedHub = errors.New("SQL storage: hub provided is invalid, new subscription was not launched")

	// ErrMalformedTopic is returned
	ErrMalformedTopic = errors.New("SQL storage: topic provided is invalid, new subscription was not launched")

	// ErrMalformedInactiveReason is returned when a subscription is ended without a reason
	ErrMalformedInactiveReason = errors.New("SQL storage: inactive reason provided is invalid, subscription was not killed")
)

// ErrUpdateFailed is returned when an update fails to touch exactly one row
type ErrUpdateFailed struct {
	numTouched int64
}

func (e ErrUpdateFailed) Error() string {
	return fmt.Sprintf("SQL storage: update touched %d rows instead of 1", e.numTouched)
}

// ErrMalformedTime is returned when the query called was expected to include a timestamp, but the timestamp was null or malformed.
type ErrMalformedTime struct {
	badTime string
}

func (e ErrMalformedTime) Error() string {
	return fmt.Sprintf("SQL storage: Stored time value {%s} could not be parsed", e.badTime)
}

// ErrNewLeaseInPast is returned when a lease extended, but the provided time is in the past.
type ErrNewLeaseInPast struct {
	badTime time.Time
}

func (e ErrNewLeaseInPast) Error() string {
	return fmt.Sprintf("SQL storage: New lease time provided {%v} was in the past", e.badTime)
}
