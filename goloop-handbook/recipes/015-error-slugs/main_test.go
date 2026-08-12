package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// body decodes the uniform error shape.
type body struct {
	Code    int    `json:"code"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func call(t *testing.T, method, path, reqBody string) (int, body) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, req)
	var b body
	_ = json.Unmarshal(rec.Body.Bytes(), &b)
	return rec.Code, b
}

// Every failure carries the slug the catalog promises, at the status it
// promises. This is the contract the frontend switches on, so it is pinned.
func TestEachFailureHasItsSlug(t *testing.T) {
	cases := []struct {
		name, method, path, reqBody string
		wantStatus                  int
		wantSlug                    string
	}{
		{"bad json", "POST", "/articles", "{not json", 400, "invalid_json"},
		{"missing slug", "POST", "/articles", `{}`, 422, "slug_required"},
		{"slug taken", "POST", "/articles", `{"slug":"existing"}`, 409, "slug_taken"},
		{"bad query", "GET", "/articles?page=-1", "", 400, "invalid_query"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, b := call(t, c.method, c.path, c.reqBody)
			if status != c.wantStatus {
				t.Errorf("status = %d, want %d", status, c.wantStatus)
			}
			if b.Error != c.wantSlug {
				t.Errorf("slug = %q, want %q", b.Error, c.wantSlug)
			}
			if b.Code != c.wantStatus {
				t.Errorf("body code = %d, want the status %d", b.Code, c.wantStatus)
			}
			if b.Message == "" {
				t.Error("no human message alongside the slug")
			}
		})
	}
}

// A body over the limit is body_too_large, told apart from invalid_json: the
// frontend must distinguish "you sent junk" from "you sent too much".
func TestBodyTooLargeHasItsOwnSlug(t *testing.T) {
	huge := `{"slug":"` + strings.Repeat("x", maxJSONBytes+1) + `"}`
	status, b := call(t, "POST", "/articles", huge)
	if status != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", status)
	}
	if b.Error != "body_too_large" {
		t.Errorf("slug = %q, want body_too_large", b.Error)
	}
}

// A good pagination request is not turned into an error, and a good create
// succeeds - the happy paths, so the slugs are not the only thing checked.
func TestHappyPaths(t *testing.T) {
	if status, _ := call(t, "GET", "/articles?page=2&per_page=50", ""); status != 200 {
		t.Errorf("valid pagination = %d, want 200", status)
	}
	if status, _ := call(t, "POST", "/articles", `{"slug":"brand-new"}`); status != 200 {
		t.Errorf("valid create = %d, want 200", status)
	}
}

// The narrative runs clean and every catalog slug appears in it.
func TestRunPlaysThrough(t *testing.T) {
	var buf bytes.Buffer
	if err := run(&buf); err != nil {
		t.Fatal(err)
	}
	for _, c := range catalog {
		if !strings.Contains(buf.String(), c.Slug) {
			t.Errorf("output missing catalog slug %q", c.Slug)
		}
	}
}
