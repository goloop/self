// Recipe 015: errors a frontend can switch on.
//
// A JSON API tells the client two things when something goes wrong: a message
// for a human, and a code for the program. A number is a poor code - 1042 says
// nothing in a log, a test or a `switch`, and mapping it to a meaning needs a
// registry nobody maintains. A slug says it plainly: "rate_limit",
// "slug_taken", "invalid_json". The frontend switches on it and looks up the
// translated message by it, in every language, without the backend ever
// sending prose the frontend has to trust.
//
// goloop/resp carries the slug beside the numeric code and the message, and
// this recipe is the small kit a real service builds on top of it:
//
//	A. one shape     - every error is {"code","error","message"}, from one
//	                   Fail helper, so no handler invents its own;
//	B. a decode kit  - Decode reads a JSON body with a size limit and answers
//	                   the right slug for each failure (invalid_json vs
//	                   body_too_large), so the frontend tells "you sent junk"
//	                   from "you sent too much";
//	C. query params  - qp reads and bounds pagination, and a bad value gets a
//	                   slug too, not a silent default that hides the bug;
//	D. the catalog   - a fixed set of slugs the frontend knows, each mapped to
//	                   a status and an i18n key.
//
// It runs against its own router with httptest, so `go run .` shows the bodies
// and `go test` pins them.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/goloop/mux"
	"github.com/goloop/qp/v2"
	"github.com/goloop/resp/v2"
)

// maxJSONBytes bounds a JSON request body: far more than any DTO needs, far
// less than a caller can use to exhaust memory. It lives on Decode, the one
// function every JSON handler calls, so the limit covers routes that do not
// exist yet without a hand-maintained list.
const maxJSONBytes = 1 << 20

// Fail writes the uniform error shape: an HTTP status, a machine-readable slug
// and a human message. Every handler in the service uses this and only this,
// so the frontend can rely on the shape.
func Fail(w http.ResponseWriter, status int, slug, message string) {
	_ = resp.Error(w, status, message, resp.WithErrorSlug(slug))
}

// Decode reads a JSON body into dst, or writes the right error and returns
// false. It tells the two failures apart on purpose: a broken body sends a
// developer to their serializer, which is the wrong place to look when the
// body was simply cut off at the size limit.
func Decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Fail(w, http.StatusRequestEntityTooLarge, "body_too_large",
				"Request body exceeds the maximum size")
			return false
		}
		Fail(w, http.StatusBadRequest, "invalid_json",
			"Request body is not valid JSON")
		return false
	}
	return true
}

// createArticle is a handler that shows every slug in one place: a bad body, a
// domain conflict, and a success.
func createArticle(taken map[string]bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Slug string `json:"slug"`
		}
		if !Decode(w, r, &in) {
			return // Decode already wrote invalid_json or body_too_large
		}
		if in.Slug == "" {
			Fail(w, http.StatusUnprocessableEntity, "slug_required",
				"A slug is required")
			return
		}
		if taken[in.Slug] {
			Fail(w, http.StatusConflict, "slug_taken",
				"That slug is already in use")
			return
		}
		taken[in.Slug] = true
		_ = resp.JSON(w, map[string]string{"slug": in.Slug})
	}
}

// listArticles reads pagination with qp and answers a slug on a bad value
// rather than silently falling back - a silent default hides the caller's bug.
//
// qp.Int returns the default on a bad value without complaint, which is the
// right behavior for an optional knob. Where a bad value should be an error,
// ParseInt returns the full Result and its .Error field carries the reason -
// so pagination that was typed wrong is a 400 with a slug, not a quiet reset
// to page 1.
func listArticles(w http.ResponseWriter, r *http.Request) {
	q := qp.New(r.URL)

	// Page and perPage are bounded: qp reads the value, and the bounds keep a
	// caller from asking for a million rows.
	pageR := q.ParseInt("page", qp.Default(1), qp.Min(1))
	perPageR := q.ParseInt("per_page", qp.Default(20), qp.Min(1), qp.Max(100))
	if pageR.Error != nil || perPageR.Error != nil {
		Fail(w, http.StatusBadRequest, "invalid_query",
			"page and per_page must be positive integers")
		return
	}

	_ = resp.JSON(w, map[string]int{
		"page": pageR.Value, "per_page": perPageR.Value,
	})
}

// newRouter wires the handlers.
func newRouter() http.Handler {
	taken := map[string]bool{"existing": true}
	r := mux.New()
	r.Post("/articles", createArticle(taken))
	r.Get("/articles", listArticles)
	return r
}

// catalog is the fixed set of slugs this service answers, the contract the
// frontend switches on. In a real repo this table is generated or shared with
// the frontend; here it doubles as example D.
var catalog = []struct {
	Slug    string
	Status  int
	I18nKey string
}{
	{"invalid_json", 400, "errors.invalid_json"},
	{"invalid_query", 400, "errors.invalid_query"},
	{"slug_required", 422, "errors.slug_required"},
	{"slug_taken", 409, "errors.slug_taken"},
	{"body_too_large", 413, "errors.body_too_large"},
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	h := newRouter()

	do := func(method, path, body string) string {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return fmt.Sprintf("%d %s", rec.Code, strings.TrimSpace(rec.Body.String()))
	}

	fmt.Fprintln(out, "A/B. one shape, right slug per failure:")
	fmt.Fprintf(out, "   bad JSON:        %s\n", do("POST", "/articles", "{not json"))
	fmt.Fprintf(out, "   missing slug:    %s\n", do("POST", "/articles", `{}`))
	fmt.Fprintf(out, "   slug taken:      %s\n", do("POST", "/articles", `{"slug":"existing"}`))
	fmt.Fprintf(out, "   created:         %s\n", do("POST", "/articles", `{"slug":"fresh"}`))

	fmt.Fprintln(out, "C. query params, a slug on a bad value:")
	fmt.Fprintf(out, "   good:            %s\n", do("GET", "/articles?page=2&per_page=50", ""))
	fmt.Fprintf(out, "   bad page:        %s\n", do("GET", "/articles?page=-1", ""))

	fmt.Fprintln(out, "D. the catalog the frontend switches on:")
	for _, c := range catalog {
		fmt.Fprintf(out, "   %-14s -> %d  %s\n", c.Slug, c.Status, c.I18nKey)
	}
	return nil
}
