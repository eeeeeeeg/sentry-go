package ingest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func validatePayload(body []byte) error {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("request body must be a JSON object")
	}
	if len(payload) == 0 {
		return fmt.Errorf("request body must not be empty")
	}

	if err := validateOptionalString(payload, "event_id"); err != nil {
		return err
	}
	if err := validateOptionalString(payload, "message"); err != nil {
		return err
	}
	if err := validateOptionalString(payload, "platform"); err != nil {
		return err
	}
	if err := validateOptionalString(payload, "release"); err != nil {
		return err
	}
	if err := validateOptionalString(payload, "environment"); err != nil {
		return err
	}

	if rawLevel, ok := payload["level"]; ok {
		level, ok := rawLevel.(string)
		if !ok {
			return fmt.Errorf("level must be a string")
		}
		if !validLevel(level) {
			return fmt.Errorf("level must be one of debug, info, warning, error, fatal")
		}
	}

	if rawTimestamp, ok := payload["timestamp"]; ok {
		timestamp, ok := rawTimestamp.(string)
		if !ok {
			return fmt.Errorf("timestamp must be an RFC3339 string")
		}
		if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
			if _, err64 := time.Parse(time.RFC3339Nano, timestamp); err64 != nil {
				return fmt.Errorf("timestamp must be an RFC3339 string")
			}
		}
	}

	if rawException, ok := payload["exception"]; ok {
		exception, ok := rawException.(map[string]any)
		if !ok {
			return fmt.Errorf("exception must be an object")
		}
		if rawValues, ok := exception["values"]; ok {
			values, ok := rawValues.([]any)
			if !ok {
				return fmt.Errorf("exception.values must be an array")
			}
			for _, rawValue := range values {
				value, ok := rawValue.(map[string]any)
				if !ok {
					return fmt.Errorf("exception.values entries must be objects")
				}
				if err := validateOptionalString(value, "type"); err != nil {
					return fmt.Errorf("exception.values.%w", err)
				}
				if err := validateOptionalString(value, "value"); err != nil {
					return fmt.Errorf("exception.values.%w", err)
				}
				if rawStacktrace, ok := value["stacktrace"]; ok {
					stacktrace, ok := rawStacktrace.(map[string]any)
					if !ok {
						return fmt.Errorf("exception.values.stacktrace must be an object")
					}
					if rawFrames, ok := stacktrace["frames"]; ok {
						if _, ok := rawFrames.([]any); !ok {
							return fmt.Errorf("exception.values.stacktrace.frames must be an array")
						}
					}
				}
			}
		}
		if err := validateOptionalString(exception, "type"); err != nil {
			return fmt.Errorf("exception.%w", err)
		}
		if err := validateOptionalString(exception, "value"); err != nil {
			return fmt.Errorf("exception.%w", err)
		}
		if rawStacktrace, ok := exception["stacktrace"]; ok {
			if _, ok := rawStacktrace.([]any); !ok {
				return fmt.Errorf("exception.stacktrace must be an array")
			}
		}
	}

	if strings.TrimSpace(stringValue(payload["message"])) == "" {
		exception, _ := payload["exception"].(map[string]any)
		if !hasExceptionSummary(exception) {
			return fmt.Errorf("message or exception is required")
		}
	}

	return nil
}

func hasExceptionSummary(exception map[string]any) bool {
	if strings.TrimSpace(stringValue(exception["type"])) != "" || strings.TrimSpace(stringValue(exception["value"])) != "" {
		return true
	}
	values, _ := exception["values"].([]any)
	for _, rawValue := range values {
		value, _ := rawValue.(map[string]any)
		if strings.TrimSpace(stringValue(value["type"])) != "" || strings.TrimSpace(stringValue(value["value"])) != "" {
			return true
		}
	}
	return false
}

func validateOptionalString(payload map[string]any, key string) error {
	if value, ok := payload[key]; ok {
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", key)
		}
	}
	return nil
}

func validLevel(level string) bool {
	switch level {
	case "debug", "info", "warning", "error", "fatal":
		return true
	default:
		return false
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
