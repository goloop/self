package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testService builds the service with cheap settings, so the suite is fast
// and does not need any environment.
func testService(t *testing.T) (*Config, *service) {
	t.Helper()
	cfg := &Config{
		JWTSecret:       "a-32-byte-or-longer-signing-secret!!",
		AuthAttempts:    3,
		AuthWindow:      time.Minute,
		AuthConcurrency: 8,
		HashConcurrency: 4,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	svc, err := newService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, svc
}

func post(h http.Handler, addr, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/signin", strings.NewReader(body))
	req.RemoteAddr = addr
	h.ServeHTTP(rec, req)
	return rec
}

// The config must refuse an unusable signing key at Validate, not at the first
// login in production. This is the difference between a rolled-back deploy and
// an outage.
func TestConfigRejectsWeakSecret(t *testing.T) {
	for _, secret := range []string{"", "too-short"} {
		c := &Config{
			JWTSecret: secret, AuthAttempts: 1, AuthWindow: time.Minute,
			HashConcurrency: 1,
		}
		if err := c.Validate(); err == nil {
			t.Errorf("Validate accepted an unusable secret %q", secret)
		}
	}
}

// The same status and slug for a wrong password and a missing account: the
// response must not reveal which addresses are registered.
func TestSignInDoesNotLeakAccountExistence(t *testing.T) {
	_, svc := testService(t)
	h := newRouter(&Config{AuthAttempts: 100, AuthWindow: time.Minute}, svc)

	wrong := post(h, "203.0.113.1:1", `{"email":"ada@example.com","password":"guess"}`)
	unknown := post(h, "203.0.113.2:1", `{"email":"nobody@example.com","password":"guess"}`)

	if wrong.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized {
		t.Fatalf("statuses differ: wrong=%d unknown=%d", wrong.Code, unknown.Code)
	}

	slug := func(rec *httptest.ResponseRecorder) string {
		var body struct{ Error string }
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return body.Error
	}
	if slug(wrong) != slug(unknown) || slug(wrong) != "invalid_credentials" {
		t.Errorf("slugs differ: wrong=%q unknown=%q", slug(wrong), slug(unknown))
	}
}

// The right password gets a token; the wrong one does not. The obvious half,
// so the security half is not the only thing checked.
func TestSignInSucceedsWithTheRightPassword(t *testing.T) {
	_, svc := testService(t)
	h := newRouter(&Config{AuthAttempts: 100, AuthWindow: time.Minute}, svc)

	rec := post(h, "203.0.113.3:1",
		`{"email":"ada@example.com","password":"correct horse battery staple"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("sign in = %d, want 200: %s", rec.Code, rec.Body)
	}
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.AccessToken == "" {
		t.Errorf("no access token in %s", rec.Body)
	}
}

// This is the test the chapter is really about. A green unit test on
// RateLimit proves the middleware works; it says nothing about whether anyone
// mounted it on the route. Here the protection is exercised through the real
// router, and - the load-bearing assertion - the test is shown to FAIL when
// the brake is removed, so it cannot pass by accident.
func TestRateLimitIsActuallyMounted(t *testing.T) {
	cfg := &Config{
		JWTSecret:    "a-32-byte-or-longer-signing-secret!!",
		AuthAttempts: 3, AuthWindow: time.Minute,
		AuthConcurrency: 8, HashConcurrency: 4,
	}
	svc, err := newService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := newRouter(cfg, svc)

	const addr = "203.0.113.9:1"
	body := `{"email":"ada@example.com","password":"guess"}`

	// The budget is three; the fourth attempt from one address must be
	// refused with 429. If newRouter forgot to mount RateLimit, every
	// attempt would be a 401 and this loop would never see a 429 - which is
	// exactly the failure the assertion below reports.
	var got429 bool
	for i := 0; i < cfg.AuthAttempts+2; i++ {
		if post(h, addr, body).Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("the rate limit is not attached to /auth: no request was " +
			"refused with 429 within the budget - a protection that is " +
			"implemented but not mounted is not a protection")
	}

	// A different address still has its own budget: the limit is per client,
	// not global.
	if code := post(h, "203.0.113.10:1", body).Code; code == http.StatusTooManyRequests {
		t.Error("a second address was refused - the limit is not per client")
	}
}

// The two refusals the chapter quotes are byte-for-byte identical bodies, so
// the frontend switches on the slug and the timing does not leak the table.
func TestRefusalBodiesAreIdentical(t *testing.T) {
	_, svc := testService(t)
	h := newRouter(&Config{AuthAttempts: 100, AuthWindow: time.Minute}, svc)

	wrong := post(h, "203.0.113.5:1", `{"email":"ada@example.com","password":"x"}`)
	unknown := post(h, "203.0.113.6:1", `{"email":"no@example.com","password":"x"}`)

	if wrong.Body.String() != unknown.Body.String() {
		t.Errorf("bodies differ:\n wrong=%s\n unknown=%s", wrong.Body, unknown.Body)
	}
	const want = `{"code":401,"error":"invalid_credentials","message":"Invalid credentials"}`
	if strings.TrimSpace(wrong.Body.String()) != want {
		t.Errorf("body = %s, want %s", strings.TrimSpace(wrong.Body.String()), want)
	}
}
