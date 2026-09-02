// Recipe 016: a catalog people edit by hand.
//
// Two YAML files arrive at this service and they are not the same kind of
// thing. One is written by a curator on the team, in a format we own. The
// other is uploaded by a user, in a format someone else defined. The
// decoder has to treat them differently, and which options it is given is
// the whole difference between a typo that is caught and a setting that
// silently never takes effect.
package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/goloop/yaml"
)

// --- The file a curator writes -------------------------------------------

// Catalog is the editorial source of truth. Every field a curator must
// state carries ",required"; every field they may leave out carries a
// "def". Between them the struct says what the file has to contain, and
// says it once, where the field is declared.
type Catalog struct {
	Version int      `yaml:"version,required"`
	Sources []Source `yaml:"sources,required"`

	// Defaults is where a curator parks an anchor to reuse further down.
	// It has to be declared even though nothing reads it: YAML has no
	// notion of a block that exists only to be referenced, so an anchor
	// holder is a key like any other, and a strict decode refuses keys
	// the struct does not know. Leaving it out is how the first curator
	// to reach for "&defaults" gets told their file is wrong.
	Defaults map[string]any `yaml:"defaults,omitempty"`
}

// Source is one entry of the catalog.
//
// The three tags do three different jobs. ",required" refuses a file that
// leaves the key out. "def" fills a key the curator did not need to think
// about. And the types say what the text has to look like: a duration is
// written the way a person writes one, a URL is parsed rather than kept as
// a string that may or may not be one.
type Source struct {
	Slug     string        `yaml:"slug,required"`
	Endpoint url.URL       `yaml:"endpoint,required"`
	Timeout  time.Duration `yaml:"timeout" def:"30s"`
	Quality  int           `yaml:"quality" def:"80"`
	Enabled  bool          `yaml:"enabled" def:"true"`
	Token    string        `yaml:"token,omitempty"`
	Note     string        `yaml:"note,omitempty"`
}

// LoadCatalog reads a file the team writes.
//
// WithStrict because a curator maintains it by hand: an unknown key there
// is nearly always a typo in a known one, and skipping it would mean the
// edit never reaches the service. WithExpandStrict because a token belongs
// in the environment rather than in a file under review, and a token that
// silently expands to nothing is worse than one that refuses to start.
func LoadCatalog(data []byte) (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(data, &c,
		yaml.WithStrict(),
		yaml.WithExpandStrict(),
	); err != nil {
		return nil, &LoadError{What: "catalog", Err: err}
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}
	return &c, nil
}

// Validate covers what a tag cannot say. A tag speaks about one field:
// that it must be present, or what it is when it is not. Rules that span
// fields, or that ask a question about the document as a whole, are the
// application's own and belong here.
func (c *Catalog) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("version %d is not supported", c.Version)
	}
	seen := make(map[string]bool, len(c.Sources))
	for _, s := range c.Sources {
		if seen[s.Slug] {
			return fmt.Errorf("slug %q appears twice", s.Slug)
		}
		seen[s.Slug] = true
	}
	return nil
}

// --- The file a user uploads ---------------------------------------------

// Manifest mirrors a format someone else defined, of which this struct is
// a subset.
type Manifest struct {
	Title string         `yaml:"title,required"`
	Pages []ManifestPage `yaml:"pages"`
}

// ManifestPage is one entry of the manifest's page list.
type ManifestPage struct {
	Title string `yaml:"title"`
	File  string `yaml:"file"`
}

// LoadManifest reads a file someone else's tool wrote.
//
// Deliberately not strict, and the reason is the opposite of the catalog's.
// The struct above is a subset of a format this importer does not own, so
// an unknown key is a field it does not need yet, not a mistake worth
// refusing the upload over. The input is untrusted, which the decoder
// already handles: alias expansion is metered and nesting is capped.
//
// No expansion either. The file came from outside, and letting it read
// this process's environment is not a feature.
func LoadManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, &LoadError{What: "manifest", Err: err}
	}
	return &m, nil
}

// --- Telling the author what is wrong ------------------------------------

// LoadError says which file failed and where, and keeps the decoder's own
// error underneath.
//
// The Unwrap method is the point of the type. Formatting the cause into a
// string and returning that would read the same and quietly end the error
// chain: errors.Is(err, yaml.ErrRequired) would stop being true one
// wrapping later, and the caller that wanted to tell "the file is
// incomplete" from "the file is wrong" would be left matching on message
// text.
type LoadError struct {
	What string // which document: "catalog", "manifest"
	Err  error  // what the decoder said
}

func (e *LoadError) Error() string { return e.What + Describe(e.Err) }

func (e *LoadError) Unwrap() error { return e.Err }

// Describe turns a decode failure into something the person who wrote the
// file can act on. They are about to go looking for the mistake, and a
// line number is the difference between finding it and guessing.
//
// A missing required key is also ErrRequired, so a caller that wants to
// treat "the file is incomplete" differently from "the file is wrong" can,
// without parsing the message.
func Describe(err error) string {
	if err == nil {
		return ""
	}
	var (
		syntax *yaml.SyntaxError
		typ    *yaml.TypeError
	)
	switch {
	case errors.As(err, &syntax):
		return fmt.Sprintf(": line %d: %s", syntax.Line, syntax.Msg)
	case errors.As(err, &typ):
		if errors.Is(err, yaml.ErrRequired) {
			return fmt.Sprintf(": line %d: incomplete: %s", typ.Line, typ.Msg)
		}
		return fmt.Sprintf(": line %d: %s", typ.Line, typ.Msg)
	}
	return ": " + err.Error()
}

// --- Writing one back out -------------------------------------------------

// Export writes a catalog back as YAML. The output is deterministic, so a
// generated file does not churn in version control.
//
// Tokens are cleared rather than written. They came from the environment
// on the way in, and an export is a file that gets attached to a ticket or
// committed to a branch; "omitempty" then leaves the key out entirely,
// which is the difference between an export that is safe to hand around
// and one that has to be scrubbed by whoever receives it.
func Export(c *Catalog) ([]byte, error) {
	out := *c
	out.Sources = slices.Clone(c.Sources)
	for i := range out.Sources {
		out.Sources[i].Token = ""
	}
	return yaml.Marshal(out)
}

// --- Demo -----------------------------------------------------------------

const curated = `
version: 1

# A curator writes the shape once and reuses it. A merge key fills gaps
# and never overwrites what an entry states for itself.
defaults: &defaults
  timeout: 45s
  quality: 90

sources:
  - <<: *defaults
    slug: weekly
    endpoint: https://example.com/feed.xml
    token: ${CATALOG_TOKEN}
    note: "costs $100 a month"

  - slug: daily
    endpoint: https://example.org/rss
    timeout: 5s
    enabled: false
`

const uploaded = `
title: Field notes
author: someone            # a key this importer does not know yet
pages:
  - title: One
    file: one.md
`

func main() {
	os.Setenv("CATALOG_TOKEN", "s3cr3t")

	c, err := LoadCatalog([]byte(curated))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, s := range c.Sources {
		fmt.Printf("%-7s %-32s timeout=%-5s quality=%d enabled=%v\n",
			s.Slug, s.Endpoint.String(), s.Timeout, s.Quality, s.Enabled)
	}
	fmt.Printf("token=%q note=%q\n\n", c.Sources[0].Token, c.Sources[0].Note)

	m, err := LoadManifest([]byte(uploaded))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("manifest %q, %d page(s), unknown keys ignored\n\n",
		m.Title, len(m.Pages))

	out, err := Export(c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Print(string(out))
}
