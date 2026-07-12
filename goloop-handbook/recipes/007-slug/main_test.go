package main

import (
	"testing"

	"github.com/goloop/scs/v2"
	"github.com/goloop/slug/v2"
	"github.com/goloop/t13n/v2"
)

func TestSlug(t *testing.T) {
	s := slug.New(slug.WithLowercase())
	taken := map[string]bool{"hello-world": true}
	if got := s.MakeUnique("Hello, World!", func(x string) bool { return taken[x] }); got != "hello-world-2" {
		t.Errorf("MakeUnique = %q", got)
	}
}

func TestTransliterate(t *testing.T) {
	latin := t13n.New().Make("Привіт")
	got := slug.New(slug.WithLowercase()).Make(latin)
	if got != "pryvit" && got != "privit" {
		t.Errorf("transliterated slug = %q", got)
	}
}

func TestCases(t *testing.T) {
	c := scs.New()
	if c.ToSnake("userAPIToken") != "user_api_token" {
		t.Errorf("snake = %q", c.ToSnake("userAPIToken"))
	}
	if c.ToKebab("userAPIToken") != "user-api-token" {
		t.Errorf("kebab = %q", c.ToKebab("userAPIToken"))
	}
}
