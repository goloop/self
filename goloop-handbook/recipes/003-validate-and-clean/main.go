// Recipe 003: validate and clean user input before you trust it.
//
// Five examples of the two halves of the job:
//
//	A. a signup form  - clean the mess and fold an email into an identity;
//	B. the is gallery - read-only predicates that answer yes or no;
//	C. the norm toolkit - Keep/Remove/DigitsOnly shape a value to a canonical form;
//	D. collect errors - validate every field at once and report all failures;
//	E. clean then check - normalize a value, then validate the canonical form.
//
// is answers "is this valid?" (read-only); norm answers "make this the
// canonical form" (cleaning and folding). Validate what must be exact; clean
// what should be forgiving.
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

// process (example A) turns a raw form into a clean account or an error.
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
	// Example A: the signup form.
	fmt.Println("A. clean and validate a signup form:")
	forms := []SignupForm{
		{Name: "  Ada​ Lovelace ", Email: "  Ada@Example.COM ", Phone: "+1 (415) 555-0132"},
		{Name: "Grace Hopper", Email: "grace@example.com"},
		{Name: "\t", Email: "grace@example.com"},
		{Name: "Alan", Email: "not-an-email"},
	}
	for i, f := range forms {
		acc, err := process(f)
		if err != nil {
			fmt.Printf("   form %d: rejected: %v\n", i+1, err)
			continue
		}
		fmt.Printf("   form %d: name=%q email=%q phone=%q\n", i+1, acc.Name, acc.Email, acc.Phone)
	}

	// Example B: the is gallery - read-only predicates for a yes/no answer,
	// handy on a query parameter or a config value.
	fmt.Println("B. is predicates (read-only, never change the value):")
	type check struct {
		name string
		ok   bool
	}
	for _, c := range []check{
		{"is.Email(a@b.co)", is.Email("a@b.co")},
		{"is.URL(https://goloop.one)", is.URL("https://goloop.one")},
		{"is.UUID(9b1deb...)", is.UUID("9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d")},
		{"is.Numeric(12345)", is.Numeric("12345")},
		{"is.Numeric(12a3)", is.Numeric("12a3")},
	} {
		fmt.Printf("   %-30s = %v\n", c.name, c.ok)
	}

	// Example C: the norm toolkit - shape a value with the character classes.
	fmt.Println("C. norm toolkit (shape a value to a canonical form):")
	fmt.Printf("   DigitsOnly(%q)  = %q\n", "+1 (415) 555-0132", norm.DigitsOnly("+1 (415) 555-0132"))
	fmt.Printf("   AlnumOnly(%q)   = %q\n", "user.name_42!", norm.AlnumOnly("user.name_42!"))
	fmt.Printf("   Remove(%q, Punct) = %q\n", "a,b;c.d", norm.Remove("a,b;c.d", norm.Punct))
	fmt.Printf("   Keep(%q, Letters) = %q\n", "Go1.24!", norm.Keep("Go1.24!", norm.Letters))

	// Example D: validate every field and collect all failures, so a form can
	// report each problem at once instead of one per round trip.
	fmt.Println("D. collect all field errors at once:")
	bad := SignupForm{Name: " ", Email: "not-an-email", Phone: "12"}
	problems := validate(bad)
	for _, field := range []string{"name", "email", "phone"} {
		if msg, ok := problems[field]; ok {
			fmt.Printf("   %-6s -> %s\n", field, msg)
		}
	}

	// Example E: clean, then check. norm produces the canonical form and is
	// verifies it - here a card number (Luhn) and an IBAN (checksum) that arrive
	// with spaces and mixed case.
	fmt.Println("E. clean then check (canonical form, then validate):")
	card, _ := norm.BankCard("4539 1488 0343 6467")
	fmt.Printf("   card %q -> is.BankCard=%v (Luhn)\n", card, is.BankCard(card))
	fmt.Printf("   card with a wrong digit -> is.BankCard=%v\n", is.BankCard("4539148803436460"))
	iban, _ := norm.IBAN("de89 3704 0044 0532 0130 00")
	fmt.Printf("   iban %q -> is.IBAN=%v (checksum)\n", iban, is.IBAN(iban))
}

// validate checks every field of a form and returns a map of field name to a
// message for each one that fails, so the caller learns all the problems in a
// single pass. An empty map means the form is valid.
func validate(f SignupForm) map[string]string {
	problems := map[string]string{}
	if norm.Clean(f.Name) == "" {
		problems["name"] = "a name is required"
	}
	if !is.Email(f.Email) {
		problems["email"] = "must be a valid email address"
	}
	if f.Phone != "" && !is.Phone(f.Phone) {
		problems["phone"] = "must be a valid phone number, or empty"
	}
	return problems
}
