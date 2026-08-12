// Recipe 011: a sign-in endpoint that survives production.
//
// The naive login handler works on day one and fails three different ways
// later: the JWT secret is missing and nobody notices until the first user
// tries to sign in; a password-guessing script hits it a thousand times a
// minute; and a burst of a hundred parallel attempts exhausts memory, because
// every Argon2id verification - wrong password or right - costs tens of
// megabytes by design.
//
// This recipe is the production pattern, assembled from the pieces:
//
//	A. boot checks   - the config refuses to start with an unusable secret
//	                   (jwt.CheckKey, TokenManager.Check);
//	B. rate limit    - per-address attempts per window (middlewares.RateLimit,
//	                   fail-closed by default);
//	C. concurrency   - a Throttle on the auth group sheds bursts at the edge,
//	                   and a semaphore around the hasher holds the memory
//	                   budget of the process;
//	D. equal cost    - the "no such account" branch burns the same hashing
//	                   work as "wrong password" (auth.BurnVerify), so response
//	                   time does not say which addresses are registered;
//	E. honest errors - every refusal carries a machine-readable slug the
//	                   frontend can switch on (resp.WithErrorSlug).
//
// Run it and the program tells the story against its own router. The wiring
// test in main_test.go is part of the pattern: it proves the protections are
// attached, not merely implemented - a green unit test on a middleware nobody
// mounted is worth nothing.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/goloop/argon2id"
	"github.com/goloop/auth"
	"github.com/goloop/env/v2"
	"github.com/goloop/jwt"
	"github.com/goloop/middlewares"
	"github.com/goloop/mux"
	"github.com/goloop/resp/v2"
)

// Config carries the knobs an operator actually turns. Struct tags cover
// presence and defaults; Validate covers what tags cannot express.
type Config struct {
	Addr string `env:"ADDR" def:":8082"`

	// JWTSecret signs access tokens. In production make it required
	// (`env:"JWT_SECRET,required"`) and set it in the environment; this
	// recipe generates an ephemeral one when it is absent so `go run .`
	// works out of the box - which also means every restart signs anew.
	JWTSecret string `env:"JWT_SECRET"`

	// The abuse budgets. Attempts per address per window is the guessing
	// brake; concurrency is the memory brake. They answer different threats
	// and both belong (see the chapter).
	AuthAttempts    int           `env:"AUTH_ATTEMPTS" def:"10"`
	AuthWindow      time.Duration `env:"AUTH_WINDOW" def:"15m"`
	AuthConcurrency int           `env:"AUTH_CONCURRENCY" def:"32"`

	// HashConcurrency caps simultaneous Argon2id operations. It is a memory
	// budget, not a speed knob: at default cost each operation holds 64 MiB,
	// so the ceiling is HashConcurrency x 64 MiB. Size it to the process,
	// not to a guess.
	HashConcurrency int `env:"HASH_CONCURRENCY" def:"4"`
}

// Validate checks what the struct tags cannot: cross-field rules and the one
// check that turns "first login fails in production" into "deploy refuses to
// boot". jwt.CheckKey is the same rule Sign and Verify apply on every call,
// exported exactly so configuration can fail first.
func (c *Config) Validate() error {
	if err := jwt.CheckKey([]byte(c.JWTSecret)); err != nil {
		return fmt.Errorf("JWT_SECRET is unusable: %w", err)
	}
	if c.AuthAttempts <= 0 || c.AuthWindow <= 0 {
		return fmt.Errorf("auth budget makes no sense: %d per %s",
			c.AuthAttempts, c.AuthWindow)
	}
	if c.HashConcurrency <= 0 {
		return fmt.Errorf("hash concurrency must be positive")
	}
	return nil
}

// Load is the canonical three steps: files into the environment, environment
// into the struct, then the rules tags cannot express.
func Load() (*Config, error) {
	_ = env.Load(".env") // optional; absent file is fine
	var c Config
	if err := env.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.JWTSecret == "" {
		// Dev convenience only. Production wants `required` on the tag and
		// a stable secret, or every deploy signs tokens nobody can verify.
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		c.JWTSecret = hex.EncodeToString(b)
		fmt.Println("note: JWT_SECRET not set; using an ephemeral dev secret")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// service is the sign-in service: a user table, a hasher behind a memory
// budget, a decoy hash for the missing-account branch, and a token manager.
type service struct {
	hasher *argon2id.Hasher
	guard  chan struct{} // the Argon2id memory budget
	decoy  string        // burned on the "no such account" branch
	tokens *auth.TokenManager
	users  map[string]string // email -> encoded hash
}

func newService(cfg *Config) (*service, error) {
	// Demo-sized Argon2id cost so the recipe runs fast; production uses the
	// defaults (64 MiB). The sizing rule is the same either way: cost per
	// operation times HashConcurrency must fit the process's memory budget.
	hasher := argon2id.New(argon2id.WithMemory(8*1024), argon2id.WithTime(1))

	// The decoy is a real hash of something nobody signs in with, made once,
	// by the same hasher with the same settings. BurnVerify spends it on the
	// branch where there is no user, so both branches cost the same.
	decoy, err := hasher.Hash([]byte("decoy-" + time.Now().String()))
	if err != nil {
		return nil, err
	}

	s := &service{
		hasher: hasher,
		guard:  make(chan struct{}, cfg.HashConcurrency),
		decoy:  decoy,
		tokens: auth.NewTokenManager([]byte(cfg.JWTSecret),
			auth.WithIssuer("handbook"), auth.WithTTL(15*time.Minute)),
		users: map[string]string{},
	}

	// TokenManager.Check answers "can this manager sign and verify at all"
	// without minting a token - the startup half of example A.
	if err := s.tokens.Check(); err != nil {
		return nil, fmt.Errorf("token manager unusable: %w", err)
	}

	// One demo account.
	stored, err := s.hash([]byte("correct horse battery staple"))
	if err != nil {
		return nil, err
	}
	s.users["ada@example.com"] = stored
	return s, nil
}

// hash runs the hasher inside the memory budget.
func (s *service) hash(password []byte) (string, error) {
	s.guard <- struct{}{}
	defer func() { <-s.guard }()
	return s.hasher.Hash(password)
}

// verify runs a verification inside the same budget. The budget lives here,
// around the operation, not on a route: the hasher is reachable from sign-in,
// password change and admin resets alike, and the memory belongs to the
// process, not to any one endpoint.
func (s *service) verify(encoded string, password []byte) error {
	s.guard <- struct{}{}
	defer func() { <-s.guard }()
	return s.hasher.Verify(encoded, password)
}

// signIn is the handler. Every refusal is the same 401 with the same slug,
// whatever actually failed - the difference between "no such account" and
// "wrong password" is exactly what an enumeration attack wants to hear.
func (s *service) signIn(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		_ = resp.Error(w, http.StatusBadRequest, "Body must be JSON",
			resp.WithErrorSlug("invalid_json"))
		return
	}

	stored, ok := s.users[strings.ToLower(in.Email)]
	if !ok {
		// The account does not exist - and this branch must cost what a
		// wrong password costs, or a stopwatch reads the user table.
		auth.BurnVerify(s.hasher, s.decoy, []byte(in.Password))
		_ = resp.Error(w, http.StatusUnauthorized, "Invalid credentials",
			resp.WithErrorSlug("invalid_credentials"))
		return
	}
	if err := s.verify(stored, []byte(in.Password)); err != nil {
		_ = resp.Error(w, http.StatusUnauthorized, "Invalid credentials",
			resp.WithErrorSlug("invalid_credentials"))
		return
	}

	token, err := s.tokens.Issue(auth.Subject{ID: in.Email})
	if err != nil {
		_ = resp.Error(w, http.StatusInternalServerError, "Cannot issue token",
			resp.WithErrorSlug("token_issue_failed"))
		return
	}
	_ = resp.JSON(w, map[string]string{"access_token": token})
}

// newRouter mounts the protections in front of the handler. The order is the
// pattern: identify the caller (RealIP), then refuse the too-frequent
// (RateLimit), then shed the too-many (Throttle), and only then do the
// expensive work.
func newRouter(cfg *Config, s *service) http.Handler {
	// Sane floors, so a partially-filled Config (a test, a half-written
	// deploy) cannot mount a limit of zero - which Throttle rightly panics
	// on. Validate is the place these are really enforced; this is a belt.
	if cfg.AuthAttempts <= 0 {
		cfg.AuthAttempts = 10
	}
	if cfg.AuthWindow <= 0 {
		cfg.AuthWindow = 15 * time.Minute
	}
	if cfg.AuthConcurrency <= 0 {
		cfg.AuthConcurrency = 32
	}

	r := mux.New()
	r.Use(middlewares.RequestID())
	r.Use(middlewares.RealIP()) // configure WithTrustedProxies behind a proxy!
	r.Use(middlewares.Recoverer())

	// The auth group carries its own brakes; the rest of the API is not
	// slowed by them.
	r.Route("/auth", func(g *mux.Router) {
		g.Use(middlewares.RateLimit(middlewares.RateLimitConfig{
			Limit:  cfg.AuthAttempts,
			Window: cfg.AuthWindow,
		}))
		g.Use(middlewares.Throttle(cfg.AuthConcurrency,
			middlewares.WithThrottleRetryAfter(2)))
		g.Post("/signin", s.signIn)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = resp.JSON(w, map[string]string{"status": "ok"})
	})
	return r
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

// run tells the story against the recipe's own router, so `go run .` shows
// the behavior without a second terminal.
func run() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	// Small budgets so the demo trips them quickly; production keeps the
	// defaults from Config.
	cfg.AuthAttempts = 3
	cfg.AuthWindow = time.Minute

	svc, err := newService(cfg)
	if err != nil {
		return err
	}
	h := newRouter(cfg, svc)

	post := func(body string) (int, string) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/signin",
			strings.NewReader(body))
		req.RemoteAddr = "203.0.113.7:40000"
		h.ServeHTTP(rec, req)
		return rec.Code, strings.TrimSpace(rec.Body.String())
	}

	fmt.Println("A. boot checks passed: jwt.CheckKey and TokenManager.Check are green")

	fmt.Println("B. sign in:")
	code, body := post(`{"email":"ada@example.com","password":"correct horse battery staple"}`)
	fmt.Printf("   right password:  %d %.40s...\n", code, body)
	code, body = post(`{"email":"ada@example.com","password":"guess"}`)
	fmt.Printf("   wrong password:  %d %s\n", code, body)
	code, body = post(`{"email":"nobody@example.com","password":"guess"}`)
	fmt.Printf("   unknown account: %d %s  <- same status, same slug, same cost\n", code, body)

	fmt.Println("C. the rate limit bites after the budget:")
	for i := 1; ; i++ {
		code, body = post(`{"email":"ada@example.com","password":"guess"}`)
		if code == http.StatusTooManyRequests {
			fmt.Printf("   attempt %d: %d %s\n", i+3, code, body)
			break
		}
		if i > 10 {
			return fmt.Errorf("the rate limit never engaged")
		}
	}

	fmt.Println("D. every refusal carried a slug the frontend can switch on")
	return nil
}
