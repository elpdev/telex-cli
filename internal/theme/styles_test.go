package theme

import "testing"

func TestBuiltInsIncludesPandoThemes(t *testing.T) {
	themes := BuiltIns()
	want := map[string]bool{
		"Muted Dark": false,
		"Phosphor":   false,
		"Miami":      false,
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
	if next := Next("Phosphor"); next.Name != "Muted Dark" {
		t.Fatalf("expected Muted Dark after Phosphor, got %q", next.Name)
	}
	if next := Next("Miami"); next.Name != "Phosphor" {
		t.Fatalf("expected Phosphor after Miami, got %q", next.Name)
	}
}

func TestBuiltInsDefineBackgrounds(t *testing.T) {
	for _, theme := range BuiltIns() {
		if theme.Background == nil {
			t.Fatalf("expected %s to define a background color", theme.Name)
		}
	}
}
