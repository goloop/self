// Recipe 006: passwords, tokens and sessions.
//
// The task: sign users in safely. Hash a password so a database leak does not
// hand out plaintext; issue a signed token an API client sends back; and keep a
// browser session in a signed cookie. Four small modules cover it:
//
//	A. auth  - hash and verify a password (PBKDF2);
//	B. auth  - issue and verify an access token (a real JWT), plus key for a
//	           short public id;
//	C. session - set and read a signed session cookie.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/goloop/auth"
	"github.com/goloop/key/v2"
	"github.com/goloop/session"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run() error {
	// Example A: password hashing. Hash stores a salted PBKDF2 digest; Verify
	// checks a candidate in constant time. The wrong password fails.
	fmt.Println("A. password hash and verify (auth PBKDF2):")
	hasher := auth.NewPBKDF2()
	encoded, err := hasher.Hash([]byte("correct horse battery staple"))
	if err != nil {
		return err
	}
	fmt.Printf("   stored hash: %.32s... (%d bytes, no plaintext)\n", encoded, len(encoded))
	fmt.Printf("   verify correct password: %v\n", hasher.Verify(encoded, []byte("correct horse battery staple")) == nil)
	fmt.Printf("   verify wrong password:   %v\n", hasher.Verify(encoded, []byte("guess")) == nil)

	// Example B: an access token. TokenManager signs a Subject into a JWT; a
	// tampered token fails verification. key mints a short public id.
	fmt.Println("B. access token issue and verify (auth + jwt), key public id:")
	tm := auth.NewTokenManager([]byte("a-32-byte-or-longer-signing-secret!!"), auth.WithIssuer("handbook"))
	token, err := tm.Issue(auth.Subject{ID: "42", Email: "ada@example.com", Roles: []string{"admin"}})
	if err != nil {
		return err
	}
	sub, err := tm.Verify(token)
	if err != nil {
		return err
	}
	fmt.Printf("   token (JWT): %.32s...\n", token)
	fmt.Printf("   verified subject: id=%s email=%s roles=%v\n", sub.ID, sub.Email, sub.Roles)
	_, tamperErr := tm.Verify(token + "x")
	fmt.Printf("   tampered token rejected: %v\n", tamperErr != nil)

	pid, _ := key.NewFixed("23456789abcdefghjkmnpqrstuvwxyz", 10)
	code, _ := pid.RandomCrypto()
	fmt.Printf("   key public id: %s\n", code)

	// Example C: a session cookie. Save signs the session into a cookie; a
	// second request reads it back. httptest stands in for a browser.
	fmt.Println("C. signed session cookie (session):")
	mgr := session.New([]byte("another-32-byte-session-signing-key!"))
	rec := httptest.NewRecorder()
	s := mgr.LoadOrNew(httptest.NewRequest(http.MethodGet, "/", nil))
	s.Subject = "42"
	s.Set("theme", "dark")
	_ = mgr.Save(rec, s)
	cookie := rec.Result().Cookies()[0]
	fmt.Printf("   set cookie %s (%.24s...)\n", cookie.Name, cookie.Value)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(cookie)
	loaded, err := mgr.Load(req2)
	if err != nil {
		return err
	}
	fmt.Printf("   loaded session: subject=%s theme=%s\n", loaded.Subject, loaded.Get("theme"))
	return nil
}
