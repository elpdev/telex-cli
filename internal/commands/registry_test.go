package commands

import "testing"

func TestRegistryRegisterAndFind(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "go-home", Title: "Home"})

	command, ok := registry.Find("go-home")
	if !ok {
		t.Fatal("expected command to be found")
	}
	if command.Title != "Home" {
		t.Fatalf("unexpected title: %q", command.Title)
	}
}

func TestRegistryFilter(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "go-home", Title: "Home", Description: "Open home", Keywords: []string{"start"}})
	registry.Register(Command{ID: "toggle-theme", Title: "Toggle Theme", Description: "Switch colors", Keywords: []string{"dark"}})

	matches := registry.Filter("dark")
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %d", len(matches))
	}
	if matches[0].ID != "toggle-theme" {
		t.Fatalf("unexpected match: %q", matches[0].ID)
	}
}

func TestRegistryListKeepsQuitLast(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "quit", Title: "Quit"})
	registry.Register(Command{ID: "themes", Title: "Themes"})
	registry.Register(Command{ID: "go-home", Title: "Home"})

	commands := registry.List()
	if commands[len(commands)-1].ID != "quit" {
		t.Fatalf("expected quit last, got %q", commands[len(commands)-1].ID)
	}
}
