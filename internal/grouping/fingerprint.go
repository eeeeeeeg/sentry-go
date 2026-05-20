package grouping

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"sentry-lite/internal/normalize"
)

var noisyValuePattern = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})|(\b\d{4,}\b)|(https?://\S+)`)

func Fingerprint(event normalize.NormalizedEvent) string {
	parts := []string{
		event.ProjectID,
		event.Platform,
		event.ExceptionType,
		normalizeMessage(firstNonEmpty(event.ExceptionValue, event.Message)),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func Title(event normalize.NormalizedEvent) string {
	if event.ExceptionType != "" && event.ExceptionValue != "" {
		return event.ExceptionType + ": " + event.ExceptionValue
	}
	if event.ExceptionType != "" {
		return event.ExceptionType
	}
	if event.Message != "" {
		return event.Message
	}
	return "Unknown error"
}

func Culprit(event normalize.NormalizedEvent) string {
	if event.RuntimeName != "" {
		return event.Platform + "/" + event.RuntimeName
	}
	return event.Platform
}

func normalizeMessage(message string) string {
	message = strings.TrimSpace(message)
	message = noisyValuePattern.ReplaceAllString(message, "?")
	return strings.Join(strings.Fields(message), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
