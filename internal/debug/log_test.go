package debug

import "testing"

func TestLogAppendsEntries(t *testing.T) {
	log := NewLog()
	log.Info("App started")
	log.Warn("Something happened")

	entries := log.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Level != "INFO" || entries[0].Message != "App started" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
}
