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

	matches := registry.Filter("dark", Context{})
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %d", len(matches))
	}
	if matches[0].ID != "toggle-theme" {
		t.Fatalf("unexpected match: %q", matches[0].ID)
	}
}

func TestParseScope(t *testing.T) {
	cases := []struct {
		in        string
		wantScope Scope
		wantRest  string
	}{
		{"send", Scope{}, "send"},
		{"drafts ", Scope{Group: GroupDrafts}, ""},
		{"drafts send", Scope{Group: GroupDrafts}, "send"},
		{"mail drafts send", Scope{Module: ModuleMail, Group: GroupDrafts}, "send"},
		{"unknown ", Scope{}, "unknown "},
	}
	for _, c := range cases {
		scope, rest := ParseScope(c.in)
		if scope != c.wantScope || rest != c.wantRest {
			t.Errorf("ParseScope(%q) = (%+v, %q), want (%+v, %q)", c.in, scope, rest, c.wantScope, c.wantRest)
		}
	}
}

func TestFilterPrefixScope(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "drafts-send", Module: ModuleMail, Group: GroupDrafts, Title: "Send"})
	registry.Register(Command{ID: "messages-archive", Module: ModuleMail, Group: GroupMessages, Title: "Archive"})
	registry.Register(Command{ID: "go-mail", Module: ModuleMail, Title: "Open Mail"})
	registry.Register(Command{ID: "themes", Module: ModuleGlobal, Title: "Themes"})

	matches := registry.Filter("drafts ", Context{})
	if len(matches) != 1 || matches[0].ID != "drafts-send" {
		t.Fatalf("expected single drafts match, got %d: %+v", len(matches), matches)
	}
}

func TestFilterContextRanking(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "themes", Module: ModuleGlobal, Title: "Themes"})
	registry.Register(Command{ID: "drafts-send", Module: ModuleMail, Group: GroupDrafts, Title: "Send"})

	// With mail as active screen, the mail command should rank first even when
	// query matches both modules' titles.
	matches := registry.Filter("", Context{ActiveScreen: ModuleMail})
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
	if matches[0].Module != ModuleMail {
		t.Fatalf("expected mail-module command first, got module %q", matches[0].Module)
	}
}

func TestFilterAvailability(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "drafts-send", Module: ModuleMail, Group: GroupDrafts, Title: "Send", Available: func(ctx Context) bool {
		return ctx.Selection != nil && ctx.Selection.IsDraft
	}})

	if got := registry.Filter("send", Context{}); len(got) != 0 {
		t.Fatalf("expected unavailable command to be filtered out, got %+v", got)
	}
	got := registry.Filter("send", Context{Selection: &Selection{IsDraft: true}})
	if len(got) != 1 {
		t.Fatalf("expected available command to appear, got %d", len(got))
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
