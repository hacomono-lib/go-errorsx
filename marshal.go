package errorsx

import (
	"encoding/json"
	"strings"
)

// MarshalJSON implements the json.Marshaler interface for Error, providing structured output for logging and APIs.
func (e *Error) MarshalJSON() ([]byte, error) {
	type jsonStack struct {
		Msg    string   `json:"msg"`
		Frames []string `json:"frames"`
	}
	type jsonCause struct {
		Msg  string `json:"msg"`
		Type string `json:"type"`
	}
	type jsonError struct {
		ID          string      `json:"id"`
		Msg         string      `json:"msg"`
		Type        ErrorType   `json:"type"`
		Status      int         `json:"status"`
		MessageData any         `json:"message_data,omitempty"`
		IsRetryable bool        `json:"is_retryable,omitempty"`
		Stacks      []jsonStack `json:"stacks,omitempty"`
		Cause       *jsonCause  `json:"cause,omitempty"`
	}

	var stacks []jsonStack
	for _, st := range e.stacks {
		jsonFrames := toStackTraceLines(st)
		if e.stackTraceCleaner != nil {
			jsonFrames = e.stackTraceCleaner(jsonFrames)
		}
		stacks = append(stacks, jsonStack{Msg: st.Msg, Frames: jsonFrames})
	}

	var cause *jsonCause
	if e.cause != nil {
		cause = &jsonCause{
			Msg:  e.cause.Error(),
			Type: causeTypeName(e.cause),
		}
	}

	return json.Marshal(jsonError{
		ID:          e.id,
		Msg:         e.msg,
		Type:        e.Type(),
		Status:      e.status,
		MessageData: e.messageData,
		IsRetryable: e.isRetryable,
		Stacks:      stacks,
		Cause:       cause,
	})
}

// causeTypeName returns the error type as a string for JSON output.
func causeTypeName(err error) string {
	if e, ok := err.(*Error); ok {
		return string(e.Type())
	}
	return "undefined"
}

// trimFunction returns the function name without the full package path.
func trimFunction(full string) string {
	if idx := strings.LastIndex(full, "/"); idx >= 0 {
		return full[idx+1:]
	}
	return full
}
