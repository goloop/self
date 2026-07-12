package main

import "testing"

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
