// Recipe 010: the whole stack, one small service.
//
// A notes API that a person owns: sign up, log in, and create and list notes
// behind a bearer token. It ties together the earlier chapters -
//
//	config (env/opt) . lifecycle (app) . observability (observe) . logging (log)
//	routing (mux) . responses (resp) . middleware (middlewares)
//	validation (norm) . auth (auth) . database (pgc-generated store)
//
// - and nothing here is a framework: each is a small package doing its job.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"goloop.one/handbook/010-whole-stack/internal/store"

	"github.com/goloop/app"
	"github.com/goloop/auth"
	"github.com/goloop/env/v2"
	"github.com/goloop/log/v2"
	"github.com/goloop/middlewares"
	"github.com/goloop/mux"
	"github.com/goloop/norm"
	"github.com/goloop/observe"
	"github.com/goloop/opt/v2"
	"github.com/goloop/resp/v2"

	_ "github.com/lib/pq"
)

// Config is read from the environment and flags (chapter 01).
type Config struct {
	Addr        string `env:"NOTES_ADDR" def:":8087" opt:"addr"`
	DatabaseURL string `env:"PGC_DATABASE_URL" opt:"db"`
	AuthSecret  string `env:"NOTES_AUTH_SECRET" def:"a-32-byte-or-longer-signing-secret!!" opt:"-"`
}

// api holds the resources the handlers share.
type api struct {
	cfg    Config
	log    *log.Logger
	db     *sql.DB
	q      *store.Queries
	hasher auth.PasswordHasher
	tm     *auth.TokenManager
	router *mux.Router
}

func main() {
	if err := run(context.Background()); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var cfg Config
	_ = env.LoadSafe()
	if err := env.Unmarshal(&cfg); err != nil {
		return err
	}
	if err := opt.UnmarshalArgs(&cfg, os.Args[1:]); err != nil {
		return err
	}

	logger := log.NewSlog("notes")
	a := &api{
		cfg:    cfg,
		hasher: auth.NewPBKDF2(),
		tm:     auth.NewTokenManager([]byte(cfg.AuthSecret), auth.WithIssuer("notes")),
	}
	a.routes()

	reg := observe.New(observe.WithService("notes"))
	reg.Check("database", func(ctx context.Context) error { return a.db.PingContext(ctx) })

	handler := middlewares.Handler(a.router,
		middlewares.RequestID(), middlewares.Recoverer(),
		middlewares.Logger(), middlewares.SecurityHeaders(),
	)
	root := http.NewServeMux()
	root.Handle("GET /healthz", reg.HealthHandler())
	root.Handle("GET /readyz", reg.ReadyHandler())
	root.Handle("/", handler)

	application := app.New("notes", app.WithLogger(logger))
	application.Use(app.HTTPServer("http", &http.Server{Addr: cfg.Addr, Handler: root}))
	application.Use(a) // the api is a component: it opens and closes the DB pool
	application.OnStart(func(context.Context) error { logger.Info("notes up", "addr", cfg.Addr); return nil })
	return application.Run(ctx)
}

// --- api as an app.Component: Start opens the DB, Stop closes it ---

func (a *api) Name() string { return "api" }

func (a *api) Start(ctx context.Context) error {
	db, err := sql.Open("postgres", a.cfg.DatabaseURL)
	if err != nil {
		return err
	}
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(c); err != nil {
		return err
	}
	a.db, a.q = db, store.New(db)
	return nil
}

func (a *api) Stop(context.Context) error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// --- routes ---

func (a *api) routes() {
	r := mux.New()
	r.Post("/v1/signup", a.signup)
	r.Post("/v1/login", a.login)
	r.Handle("POST /v1/notes", a.tm.Protect(http.HandlerFunc(a.createNote)))
	r.Handle("GET /v1/notes", a.tm.Protect(http.HandlerFunc(a.listNotes)))
	a.router = r
}

func (a *api) signup(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	_ = json.NewDecoder(r.Body).Decode(&in)
	email, ok := norm.EmailFold(in.Email)
	if !ok || len(in.Password) < 8 {
		_ = resp.Error(w, http.StatusBadRequest, "valid email and 8+ char password required")
		return
	}
	hash, _ := a.hasher.Hash([]byte(in.Password))
	u, err := a.q.CreateUser(r.Context(), email, hash)
	if err != nil {
		_ = resp.Error(w, http.StatusConflict, "email already registered")
		return
	}
	_ = resp.Created(w, "/v1/me", resp.R{"token": a.token(u), "email": u.Email})
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	_ = json.NewDecoder(r.Body).Decode(&in)
	email, _ := norm.EmailFold(in.Email)
	u, err := a.q.UserByEmail(r.Context(), email)
	if err != nil || a.hasher.Verify(u.PasswordHash, []byte(in.Password)) != nil {
		_ = resp.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	_ = resp.JSON(w, resp.R{"token": a.token(u)})
}

func (a *api) createNote(w http.ResponseWriter, r *http.Request) {
	uid := a.subjectID(r)
	var in struct{ Title string }
	_ = json.NewDecoder(r.Body).Decode(&in)
	title := norm.Clean(in.Title)
	if title == "" {
		_ = resp.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	n, err := a.q.CreateNote(r.Context(), uid, title)
	if err != nil {
		_ = resp.Error(w, http.StatusInternalServerError, "could not create note")
		return
	}
	_ = resp.Created(w, "/v1/notes/"+strconv.FormatInt(n.ID, 10), n)
}

func (a *api) listNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := a.q.NotesByUser(r.Context(), a.subjectID(r))
	if err != nil {
		_ = resp.Error(w, http.StatusInternalServerError, "could not list notes")
		return
	}
	_ = resp.JSON(w, resp.R{"notes": notes})
}

func (a *api) token(u store.User) string {
	t, _ := a.tm.Issue(auth.Subject{ID: strconv.FormatInt(u.ID, 10), Email: u.Email})
	return t
}

func (a *api) subjectID(r *http.Request) int64 {
	sub, _ := auth.SubjectFrom(r.Context())
	id, _ := strconv.ParseInt(sub.ID, 10, 64)
	return id
}
