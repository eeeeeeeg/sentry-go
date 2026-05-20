package normalize

import "testing"

func TestScrubFiltersNestedSensitiveKeys(t *testing.T) {
	got := Scrub(map[string]any{
		"headers": map[string]any{
			"Authorization": "Bearer abc",
		},
		"user": map[string]any{
			"name": "Ada",
		},
	}).(map[string]any)

	headers := got["headers"].(map[string]any)
	if headers["Authorization"] != redacted {
		t.Fatalf("Authorization = %v", headers["Authorization"])
	}
	user := got["user"].(map[string]any)
	if user["name"] != "Ada" {
		t.Fatalf("user.name = %v", user["name"])
	}
}
