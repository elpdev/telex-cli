package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Error struct {
	StatusCode int
	Message    string
	Details    map[string][]string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("request failed with status %d", e.StatusCode)
	}
	return e.Message
}

func parseError(status int, body []byte) error {
	var envelope Envelope[json.RawMessage]
	if err := json.Unmarshal(body, &envelope); err == nil {
		message := strings.TrimSpace(envelope.Error)
		if message != "" || len(envelope.Details) > 0 {
			return &Error{StatusCode: status, Message: message, Details: envelope.Details}
		}
	}
	return &Error{StatusCode: status, Message: strings.TrimSpace(string(body))}
}
