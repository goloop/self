package main

import (
	"testing"

	"github.com/goloop/is/v2"
	"github.com/goloop/norm"
)

func TestProcessCleansAndValidates(t *testing.T) {
	acc, err := process(SignupForm{
		Name:  "  Ada​ Lovelace ",
		Email: "  Ada@Example.COM ",
		Phone: "+1 (415) 555-0132",
	})
	if err != nil {
		t.Fatal(err)
	}
	if acc.Name != "Ada Lovelace" {
		t.Errorf("name = %q, want %q", acc.Name, "Ada Lovelace")
	}
	if acc.Email != "ada@example.com" {
		t.Errorf("email = %q, want folded lower case", acc.Email)
	}
	if acc.Phone != "+14155550132" {
		t.Errorf("phone = %q, want E.164", acc.Phone)
	}
}

func TestProcessRejects(t *testing.T) {
	if _, err := process(SignupForm{Name: "\t", Email: "a@b.co"}); err != ErrEmptyName {
		t.Errorf("empty name: got %v", err)
	}
	if _, err := process(SignupForm{Name: "X", Email: "nope"}); err != ErrInvalidEmail {
		t.Errorf("bad email: got %v", err)
	}
}

func TestIsGallery(t *testing.T) {
	if !is.Email("a@b.co") || !is.URL("https://goloop.one") || !is.Numeric("12345") {
		t.Error("expected valid inputs to pass")
	}
	if is.Numeric("12a3") {
		t.Error("is.Numeric should reject letters")
	}
}

func TestNormToolkit(t *testing.T) {
	if got := norm.DigitsOnly("+1 (415) 555-0132"); got != "14155550132" {
		t.Errorf("DigitsOnly = %q", got)
	}
	if got := norm.AlnumOnly("user.name_42!"); got != "username42" {
		t.Errorf("AlnumOnly = %q", got)
	}
	if got := norm.Keep("Go1.24!", norm.Letters); got != "Go" {
		t.Errorf("Keep Letters = %q", got)
	}
}
