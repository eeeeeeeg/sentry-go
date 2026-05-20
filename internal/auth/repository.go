package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMissingBearerToken = errors.New("authentication required")
	ErrInvalidBearerToken = errors.New("invalid auth token")
	ErrMissingScope       = errors.New("missing required scope")
)

type Repository struct {
	db *pgxpool.Pool
}

type Token struct {
	ID             string
	OrganizationID string
	Scopes         []string
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Authenticate(ctx context.Context, authorization string, allowedScopes ...string) (Token, error) {
	rawToken, err := bearerToken(authorization)
	if err != nil {
		return Token{}, err
	}
	tokenHash := hashToken(rawToken)

	var token Token
	var storedHash string
	var expiresAt *time.Time
	err = r.db.QueryRow(ctx, `
SELECT id::text, organization_id::text, token_hash, scopes, expires_at
FROM api_tokens
WHERE token_hash = $1
LIMIT 1`, tokenHash).Scan(&token.ID, &token.OrganizationID, &storedHash, &token.Scopes, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Token{}, ErrInvalidBearerToken
	}
	if err != nil {
		return Token{}, fmt.Errorf("authenticate token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(tokenHash)) != 1 {
		return Token{}, ErrInvalidBearerToken
	}
	if expiresAt != nil && time.Now().UTC().After(expiresAt.UTC()) {
		return Token{}, ErrInvalidBearerToken
	}
	if !hasAnyScope(token.Scopes, allowedScopes...) {
		return Token{}, ErrMissingScope
	}
	_, _ = r.db.Exec(ctx, `UPDATE api_tokens SET last_used_at = now() WHERE id = $1::uuid`, token.ID)
	return token, nil
}

func bearerToken(authorization string) (string, error) {
	authorization = strings.TrimSpace(authorization)
	if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return "", ErrMissingBearerToken
	}
	token := strings.TrimSpace(authorization[len("Bearer "):])
	if token == "" {
		return "", ErrMissingBearerToken
	}
	return token, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func hasAnyScope(actual []string, allowed ...string) bool {
	if len(allowed) == 0 {
		return true
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, scope := range actual {
		actualSet[strings.TrimSpace(scope)] = struct{}{}
	}
	for _, scope := range allowed {
		if _, ok := actualSet[scope]; ok {
			return true
		}
	}
	return false
}
