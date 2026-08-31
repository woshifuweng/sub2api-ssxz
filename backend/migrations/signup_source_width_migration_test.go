package migrations

import (
	"strings"
	"testing"
)

func TestSignupSourceWidthMigrationConvergesUpgradePaths(t *testing.T) {
	sql, err := FS.ReadFile("235_align_signup_source_width.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(string(sql)), " "))
	if !strings.Contains(normalized, "alter column signup_source type varchar(32)") {
		t.Fatalf("migration must widen users.signup_source to varchar(32): %s", normalized)
	}
}
