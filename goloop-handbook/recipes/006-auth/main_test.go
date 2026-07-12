package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goloop/auth"
	"github.com/goloop/session"
)

func TestPassword(t *testing.T) {
	h := auth.NewPBKDF2()
	enc, err := h.Hash([]byte("pw"))
	if err != nil {
		t.Fatal(err)
	}
	if h.Verify(enc, []byte("pw")) != nil {
		t.Error("correct password rejected")
	}
	if h.Verify(enc, []byte("nope")) == nil {
		t.Error("wrong password accepted")
	}
}

func TestToken(t *testing.T) {
	tm := auth.NewTokenManager([]byte("a-32-byte-or-longer-signing-secret!!"))
	tok, err := tm.Issue(auth.Subject{ID: "42", Roles: []string{"admin"}})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := tm.Verify(tok)
	if err != nil || sub.ID != "42" {
		t.Fatalf("verify: %v %+v", err, sub)
	}
	if _, err := tm.Verify(tok + "x"); err == nil {
		t.Error("tampered token accepted")
	}
}

func TestSession(t *testing.T) {
	mgr := session.New([]byte("another-32-byte-session-signing-key!"))
	rec := httptest.NewRecorder()
	s := mgr.LoadOrNew(httptest.NewRequest(http.MethodGet, "/", nil))
	s.Subject = "7"
	_ = mgr.Save(rec, s)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(rec.Result().Cookies()[0])
	loaded, err := mgr.Load(req)
	if err != nil || loaded.Subject != "7" {
		t.Fatalf("session round-trip: %v %+v", err, loaded)
	}
}
