package system

import (
	"strings"
	"testing"
)

func TestMarshalAuditData_DropsCredentialFieldsOnly(t *testing.T) {
	input := map[string]any{
		"id":       int64(9007199254740993),
		"password": "$2a$10$hash",
		"tel":      "18712345678",
		"email":    "admin@example.com",
		"profile": map[string]any{
			"refresh_token": "secret-token",
			"name":          "operator",
		},
	}

	got := marshalAuditData(input)
	for _, leaked := range []string{"password", "$2a$10$hash", "refresh_token", "secret-token"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("credential value %q leaked in audit json: %s", leaked, got)
		}
	}
	for _, kept := range []string{"18712345678", "admin@example.com", "operator", "9007199254740993"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("expected value %q to be kept in audit json: %s", kept, got)
		}
	}
}
