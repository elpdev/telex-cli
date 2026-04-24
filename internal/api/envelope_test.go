package api

import "testing"

func TestDecodeEnvelope(t *testing.T) {
	envelope, err := DecodeEnvelope[struct {
		Name string `json:"name"`
	}]([]byte(`{"data":{"name":"inbox"},"meta":{"page":2,"per_page":25,"total_count":50}}`))
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Name != "inbox" {
		t.Fatalf("name = %q", envelope.Data.Name)
	}
	pagination := DecodePagination(envelope.Meta)
	if pagination == nil || pagination.Page != 2 || pagination.PerPage != 25 || pagination.TotalCount != 50 {
		t.Fatalf("pagination = %#v", pagination)
	}
}

func TestDecodePaginationIgnoresIncompleteMeta(t *testing.T) {
	if got := DecodePagination(map[string]any{"page": 1}); got != nil {
		t.Fatalf("pagination = %#v", got)
	}
}
