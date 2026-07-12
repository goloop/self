// Recipe 003: validate and clean user input before you trust it.
//
// The task: a signup form arrives with human mess - stray spaces, mixed case,
// a zero-width character pasted from a chat app, a phone number with dashes.
// Two modules cover the two halves of the job: is answers "is this valid?"
// (read-only predicates) and norm answers "make this the canonical form"
// (cleaning and folding). Validate what must be exact; clean what should be
// forgiving.
package main

import (
	"errors"
	"fmt"

	"github.com/goloop/is/v2"
	"github.com/goloop/norm"
)

// SignupForm is the raw, untrusted input.
type SignupForm struct {
	Name  string
	Email string
	Phone string
}

// Account is the clean, validated result that the rest of the program can trust.
type Account struct {
	Name  string // scrubbed of invisible/control bytes, whitespace collapsed
	Email string // a case-insensitive identity key
	Phone string // E.164, or empty when not given
}

// ErrInvalidEmail is returned when the email cannot be made valid.
var ErrInvalidEmail = errors.New("a valid email is required")

// ErrEmptyName is returned when the name cleans to nothing.
var ErrEmptyName = errors.New("a name is required")

// process turns a raw form into a clean account or an error.
func process(f SignupForm) (Account, error) {
	// Clean removes invisible/control characters, collapses whitespace and
	// trims - the safe one-call cleanup for a free-text field.
	name := norm.Clean(f.Name)
	if name == "" {
		return Account{}, ErrEmptyName
	}

	// EmailFold validates and normalizes the email to a lower-cased identity,
	// so "Ada@Example.COM" and "ada@example.com" are the same account.
	email, ok := norm.EmailFold(f.Email)
	if !ok {
		return Account{}, ErrInvalidEmail
	}

	// Phone is optional. E164 validates and normalizes to +<digits>; an empty
	// or invalid phone is simply dropped rather than rejecting the signup.
	phone := ""
	if f.Phone != "" {
		if e164, ok := norm.E164(f.Phone); ok {
			phone = e164
		}
	}

	return Account{Name: name, Email: email, Phone: phone}, nil
}

func main() {
	forms := []SignupForm{
		{Name: "  Ada​ Lovelace ", Email: "  Ada@Example.COM ", Phone: "+1 (415) 555-0132"},
		{Name: "Grace Hopper", Email: "grace@example.com", Phone: ""},
		{Name: "\t", Email: "grace@example.com"},
		{Name: "Alan", Email: "not-an-email"},
	}
	for i, f := range forms {
		acc, err := process(f)
		if err != nil {
			fmt.Printf("form %d: rejected: %v\n", i+1, err)
			continue
		}
		fmt.Printf("form %d: name=%q email=%q phone=%q\n", i+1, acc.Name, acc.Email, acc.Phone)
	}

	// is is the read-only half: use it directly when you only need a yes/no,
	// for example on a query parameter or a config value.
	fmt.Println("---")
	fmt.Printf("is.Email(\"a@b.co\") = %v\n", is.Email("a@b.co"))
	fmt.Printf("is.Numeric(\"12345\") = %v\n", is.Numeric("12345"))
}
