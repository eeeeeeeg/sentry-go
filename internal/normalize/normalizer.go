package normalize

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"sentry-lite/pkg/envelope"
	"sentry-lite/pkg/event"
)

var ErrMissingPayload = errors.New("raw event payload is missing")

type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

func (n *Normalizer) Normalize(raw RawEventMessage) (NormalizedEvent, error) {
	if len(raw.Payload) == 0 {
		return NormalizedEvent{}, ErrMissingPayload
	}

	var payload map[string]any
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return NormalizedEvent{}, fmt.Errorf("decode payload for scrub: %w", err)
	}
	scrubbedBytes, err := json.Marshal(Scrub(payload))
	if err != nil {
		return NormalizedEvent{}, fmt.Errorf("encode scrubbed payload: %w", err)
	}

	var env envelope.Envelope
	if err := json.Unmarshal(scrubbedBytes, &env); err != nil {
		return NormalizedEvent{}, fmt.Errorf("decode envelope: %w", err)
	}

	now := time.Now().UTC()
	timestamp := env.Timestamp
	if timestamp.IsZero() {
		timestamp = raw.ReceivedAt
	}
	if timestamp.IsZero() {
		timestamp = now
	}
	receivedAt := raw.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = now
	}

	level := strings.ToLower(strings.TrimSpace(env.Level))
	if level == "" {
		level = string(event.LevelError)
	}
	if !event.Level(level).Valid() {
		level = string(event.LevelError)
	}

	eventID := strings.TrimSpace(env.EventID)
	if eventID == "" {
		eventID = newUUID()
	} else {
		eventID = canonicalEventID(eventID)
	}

	sdkName := firstNonEmpty(env.SDK.Name, anyStringMap(payload, "sdk", "name"), raw.SDKName)
	sdkVersion := firstNonEmpty(env.SDK.Version, anyStringMap(payload, "sdk", "version"), raw.SDKVersion)
	message := firstNonEmpty(env.Message, anyStringMap(payload, "logentry", "formatted"), anyStringMap(payload, "logentry", "message"))
	exceptionType, exceptionValue := exceptionSummary(payload, env.Exception)
	if message == "" {
		message = firstNonEmpty(exceptionValue, exceptionType)
	}

	userID := ""
	if env.User != nil {
		userID = anyString(env.User["id"])
		if userID == "" {
			userID = anyString(env.User["email"])
		}
	}

	return NormalizedEvent{
		EventID:        eventID,
		OrganizationID: raw.OrganizationID,
		ProjectID:      raw.ProjectID,
		ProjectKeyID:   raw.ProjectKeyID,
		Timestamp:      timestamp.UTC(),
		ReceivedAt:     receivedAt.UTC(),
		Platform:       firstNonEmpty(env.Platform, "unknown"),
		RuntimeName:    env.Runtime.Name,
		RuntimeVersion: env.Runtime.Version,
		SDKName:        sdkName,
		SDKVersion:     sdkVersion,
		Level:          level,
		Message:        message,
		ExceptionType:  exceptionType,
		ExceptionValue: exceptionValue,
		Release:        env.Release,
		Environment:    firstNonEmpty(env.Environment, "production"),
		UserID:         userID,
		Tags:           env.Tags,
		Contexts:       env.Contexts,
		RawEvent:       scrubbedBytes,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func anyStringMap(payload map[string]any, keys ...string) string {
	var current any = payload
	for _, key := range keys {
		mapped, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = mapped[key]
	}
	return anyString(current)
}

func canonicalEventID(eventID string) string {
	eventID = strings.TrimSpace(eventID)
	if len(eventID) != 32 {
		return eventID
	}
	for _, r := range eventID {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return eventID
		}
	}
	lower := strings.ToLower(eventID)
	return fmt.Sprintf("%s-%s-%s-%s-%s", lower[0:8], lower[8:12], lower[12:16], lower[16:20], lower[20:32])
}

func exceptionSummary(payload map[string]any, fallback *envelope.Exception) (string, string) {
	exception, _ := payload["exception"].(map[string]any)
	values, _ := exception["values"].([]any)
	for i := len(values) - 1; i >= 0; i-- {
		value, _ := values[i].(map[string]any)
		exceptionType := strings.TrimSpace(anyString(value["type"]))
		exceptionValue := strings.TrimSpace(anyString(value["value"]))
		if exceptionType != "" || exceptionValue != "" {
			return exceptionType, exceptionValue
		}
	}
	if fallback != nil {
		return strings.TrimSpace(fallback.Type), strings.TrimSpace(fallback.Value)
	}
	return "", ""
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}
