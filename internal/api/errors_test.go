package api

import (
	"fmt"
	"net/http"
	"testing"
)

func TestIsStatusMatchesAPIErrorStatus(t *testing.T) {
	err := fmt.Errorf("delete draft: %w", &Error{StatusCode: http.StatusNotFound, Message: "missing"})
	if !IsStatus(err, http.StatusNotFound) {
		t.Fatal("expected wrapped not found API error to match")
	}
	if IsStatus(err, http.StatusUnauthorized) {
		t.Fatal("unexpected status match")
	}
}
