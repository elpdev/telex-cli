package theme

import "testing"

func TestBuiltInsIncludesSharedThemes(t *testing.T) {
	themes := BuiltIns()
	want := map[string]bool{
		"Phosphor":          false,
		"Dracula":           false,
		"Tokyo Night":       false,
		"Catppuccin Mocha": false,
		"Nord":              false,
		"Gruvbox Dark":      false,
		"Monokai":           false,
	}

	for _, theme := range themes {
		if _, ok := want[theme.Name]; ok {
			want[theme.Name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("expected built-in theme %q", name)
		}
	}
}

func TestNextCyclesThemes(t *testing.T) {
	if next := Next("Phosphor"); next.Name != "Dracula" {
		t.Fatalf("expected Dracula after Phosphor, got %q", next.Name)
	}
	if next := Next("Monokai"); next.Name != "Phosphor" {
		t.Fatalf("expected Phosphor after Monokai, got %q", next.Name)
	}
}

func TestBuiltInsDefineBackgrounds(t *testing.T) {
	for _, theme := range BuiltIns() {
		if theme.Background == nil {
			t.Fatalf("expected %s to define a background color", theme.Name)
		}
	}
}
