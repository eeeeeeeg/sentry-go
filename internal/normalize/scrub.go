package normalize

import "strings"

var sensitiveKeys = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"set-cookie":    {},
	"password":      {},
	"passwd":        {},
	"token":         {},
	"secret":        {},
	"access_token":  {},
	"refresh_token": {},
}

const redacted = "[Filtered]"

func Scrub(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveKey(key) {
				out[key] = redacted
				continue
			}
			out[key] = Scrub(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, Scrub(item))
		}
		return out
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if _, ok := sensitiveKeys[normalized]; ok {
		return true
	}
	return strings.Contains(normalized, "token") || strings.Contains(normalized, "secret")
}
