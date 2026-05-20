package dsn

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrEmptyDSN       = errors.New("dsn is empty")
	ErrInvalidDSN     = errors.New("dsn is invalid")
	ErrMissingKey     = errors.New("dsn public key is missing")
	ErrMissingProject = errors.New("dsn project id is missing")
)

type DSN struct {
	Scheme    string
	Host      string
	PublicKey string
	ProjectID string
}

func Parse(raw string) (DSN, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DSN{}, ErrEmptyDSN
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return DSN{}, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return DSN{}, ErrInvalidDSN
	}

	publicKey := parsed.User.Username()
	if publicKey == "" {
		return DSN{}, ErrMissingKey
	}

	projectID := strings.Trim(parsed.Path, "/")
	if projectID == "" || strings.Contains(projectID, "/") {
		return DSN{}, ErrMissingProject
	}

	return DSN{
		Scheme:    parsed.Scheme,
		Host:      parsed.Host,
		PublicKey: publicKey,
		ProjectID: projectID,
	}, nil
}
