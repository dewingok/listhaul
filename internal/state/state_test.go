package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := New(path)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := store.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	store.Mark("INBOX", 42)
	if store.Seen("INBOX", 42) != true {
		t.Fatal("expected uid to be marked seen")
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := New(path)
	if err != nil {
		t.Fatalf("New() reload error = %v", err)
	}
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load() reload error = %v", err)
	}
	if !reloaded.Seen("INBOX", 42) {
		t.Fatal("expected persisted uid")
	}
}

func TestExpandPathHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	got, err := expandPath("~/.listhaul/state.json")
	if err != nil {
		t.Fatalf("expandPath() error = %v", err)
	}
	want := filepath.Join(home, ".listhaul", "state.json")
	if got != want {
		t.Fatalf("expandPath() = %q, want %q", got, want)
	}
}
