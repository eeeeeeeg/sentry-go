package ingest

import (
	"errors"
	"net/http"
	"strings"

	"sentry-lite/pkg/dsn"
)

var ErrMissingPublicKey = errors.New("dsn public key is required")

func extractPublicKey(r *http.Request) (string, error) {
	if key := strings.TrimSpace(r.Header.Get("X-Sentry-Key")); key != "" {
		return key, nil
	}
	if key := strings.TrimSpace(r.URL.Query().Get("sentry_key")); key != "" {
		return key, nil
	}
	if rawDSN := strings.TrimSpace(r.Header.Get("X-DSN")); rawDSN != "" {
		parsed, err := dsn.Parse(rawDSN)
		if err != nil {
			return "", err
		}
		return parsed.PublicKey, nil
	}

	if sentryAuth := strings.TrimSpace(r.Header.Get("X-Sentry-Auth")); sentryAuth != "" {
		return parseSentryAuth(sentryAuth)
	}

	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return "", ErrMissingPublicKey
	}

	if strings.HasPrefix(auth, "DSN ") {
		parsed, err := dsn.Parse(strings.TrimSpace(strings.TrimPrefix(auth, "DSN ")))
		if err != nil {
			return "", err
		}
		return parsed.PublicKey, nil
	}

	if strings.HasPrefix(auth, "Sentry ") {
		return parseSentryAuth(auth)
	}

	return "", ErrMissingPublicKey
}

func parseSentryAuth(auth string) (string, error) {
	value := strings.TrimSpace(strings.TrimPrefix(auth, "Sentry "))
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == "sentry_key" {
			val = strings.Trim(strings.TrimSpace(val), `"`)
			if val == "" {
				return "", ErrMissingPublicKey
			}
			return val, nil
		}
	}
	return "", ErrMissingPublicKey
}
