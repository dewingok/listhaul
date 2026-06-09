package config

import (
	"os"
	"path/filepath"
	"testing"
)

const validConfig = `
[poll]
interval = "10m"
lookback = "60m"

[account]
host = "imap.example.com"
username = "user@example.com"
password_env = "LISTHAUL_IMAP_PASSWORD"

[actions]
enabled = ["fileinto", "stop"]

[[rules]]
name = "newsletters"
[rules.if]
header = { names = ["List-Id"], match = "contains", values = ["lists.example.com"] }

[[rules.then]]
action = "fileinto"
folder = "INBOX/Newsletters"

[[rules.then]]
action = "stop"
`

func TestLoadValidConfig(t *testing.T) {
	path := writeTempConfig(t, validConfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Poll.Interval.Duration != DefaultPollInterval {
		t.Fatalf("interval = %v, want %v", cfg.Poll.Interval.Duration, DefaultPollInterval)
	}
	if cfg.Account.Port != 993 {
		t.Fatalf("port = %d, want 993", cfg.Account.Port)
	}
}

func TestLoadRejectsDisabledAction(t *testing.T) {
	config := validConfig + `
[[rules]]
name = "bad"
[rules.if]
header = { names = ["Subject"], match = "contains", values = ["test"] }

[[rules.then]]
action = "discard"
`
	path := writeTempConfig(t, config)
	if _, err := Load(path); err == nil {
		t.Fatal("expected disabled action error")
	}
}

func TestLoadRejectsEmptyTest(t *testing.T) {
	config := `
[account]
host = "imap.example.com"
username = "user@example.com"
password_env = "LISTHAUL_IMAP_PASSWORD"

[[rules]]
[rules.if]

[[rules.then]]
action = "stop"
`
	path := writeTempConfig(t, config)
	if _, err := Load(path); err == nil {
		t.Fatal("expected empty test error")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "listhaul.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
