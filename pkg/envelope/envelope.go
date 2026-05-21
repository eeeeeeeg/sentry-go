package envelope

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Envelope struct {
	EventID     string            `json:"event_id"`
	Timestamp   FlexibleTime      `json:"timestamp"`
	Platform    string            `json:"platform"`
	Runtime     Runtime           `json:"runtime"`
	SDK         SDK               `json:"sdk"`
	Level       string            `json:"level"`
	Message     string            `json:"message"`
	Exception   *Exception        `json:"exception,omitempty"`
	Release     string            `json:"release,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	User        map[string]any    `json:"user,omitempty"`
	Contexts    map[string]any    `json:"contexts,omitempty"`
	Breadcrumbs []Breadcrumb      `json:"breadcrumbs,omitempty"`
	Fingerprint []string          `json:"fingerprint,omitempty"`
}

type Runtime struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type SDK struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Exception struct {
	Type       string       `json:"type"`
	Value      string       `json:"value"`
	Stacktrace []StackFrame `json:"stacktrace,omitempty"`
}

type StackFrame struct {
	Filename string `json:"filename,omitempty"`
	Function string `json:"function,omitempty"`
	Module   string `json:"module,omitempty"`
	LineNo   int    `json:"lineno,omitempty"`
	ColNo    int    `json:"colno,omitempty"`
	InApp    bool   `json:"in_app,omitempty"`
}

type Breadcrumb struct {
	Timestamp FlexibleTime   `json:"timestamp,omitempty"`
	Type      string         `json:"type,omitempty"`
	Category  string         `json:"category,omitempty"`
	Level     string         `json:"level,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

type FlexibleTime struct {
	time.Time
}

func (t *FlexibleTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		parsed, err := parseFlexibleTimeString(text)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	}

	if parsed, ok, err := parseNumericTimestamp(data); ok || err != nil {
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	}

	return fmt.Errorf("timestamp must be an RFC3339 string or Unix timestamp number")
}

func parseNumericTimestamp(data []byte) (time.Time, bool, error) {
	text := strings.TrimSpace(string(data))
	if text == "" || text[0] == '"' {
		return time.Time{}, false, nil
	}

	parts := strings.SplitN(text, ".", 2)
	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, false, nil
	}
	if seconds < 0 {
		return time.Time{}, true, fmt.Errorf("timestamp must be an RFC3339 string or Unix timestamp number")
	}

	var nanos int64
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > 9 {
			fraction = fraction[:9]
		}
		for len(fraction) < 9 {
			fraction += "0"
		}
		nanos, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return time.Time{}, true, fmt.Errorf("timestamp must be an RFC3339 string or Unix timestamp number")
		}
	}

	return time.Unix(seconds, nanos).UTC(), true, nil
}

func parseFlexibleTimeString(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp must be an RFC3339 string or Unix timestamp number")
	}
	return parsed.UTC(), nil
}
