package frontmatter

import (
	"strings"
	"testing"
)

func TestParseFrontmatterDocument(t *testing.T) {
	doc, err := Parse("---\nname: Alice\nphone: \"+1 555: 123\"\n# comment\n---\n\nBody")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Fields["name"] != "Alice" || doc.Fields["phone"] != "+1 555: 123" || doc.Body != "Body" {
		t.Fatalf("doc = %#v", doc)
	}
}

func TestParsePlainDocument(t *testing.T) {
	doc, err := Parse("Body only")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Fields) != 0 || doc.Body != "Body only" {
		t.Fatalf("doc = %#v", doc)
	}
}

func TestRenderFrontmatterDocument(t *testing.T) {
	out := Render(map[string]string{"name": "Alice", "phone": "+1 555: 123"}, "Body")
	for _, want := range []string{"---\n", "name: Alice\n", "phone: \"+1 555: 123\"\n", "---\n\nBody"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered document missing %q in %q", want, out)
		}
	}
}

func TestRenderFrontmatterDocumentWithOrder(t *testing.T) {
	out := RenderWithOrder(map[string]string{"name": "Alice", "contact_type": "person"}, []string{"contact_type", "name"}, "Body")
	if !strings.HasPrefix(out, "---\ncontact_type: person\nname: Alice\n") {
		t.Fatalf("out = %q", out)
	}
}

func TestParseRejectsUnclosedFrontmatter(t *testing.T) {
	if _, err := Parse("---\nname: Alice\nBody"); err == nil {
		t.Fatal("expected error")
	}
}
