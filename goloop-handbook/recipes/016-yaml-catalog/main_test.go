package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/goloop/yaml"
)

const minimal = `
version: 1
sources:
  - slug: only
    endpoint: https://example.com/x
`

func TestDefaultsFillWhatTheCuratorLeftOut(t *testing.T) {
	c, err := LoadCatalog([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	s := c.Sources[0]
	if s.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want the default", s.Timeout)
	}
	if s.Quality != 80 {
		t.Errorf("quality = %d, want the default", s.Quality)
	}
	if !s.Enabled {
		t.Error("enabled should default to true")
	}
}

func TestTheDocumentBeatsTheDefault(t *testing.T) {
	src := minimal + "    timeout: 5s\n    quality: 10\n"
	c, err := LoadCatalog([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].Timeout != 5*time.Second || c.Sources[0].Quality != 10 {
		t.Errorf("a default overruled the file: %+v", c.Sources[0])
	}
}

// An explicit null is a decision, and a default that undid it would be
// undoing the curator.
func TestExplicitNullIsNotAbsence(t *testing.T) {
	src := minimal + "    quality: null\n"
	c, err := LoadCatalog([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].Quality != 0 {
		t.Errorf("quality = %d, want 0: null was written on purpose",
			c.Sources[0].Quality)
	}
}

func TestRequiredNamesWhatIsMissing(t *testing.T) {
	src := "version: 1\nsources:\n  - endpoint: https://example.com/x\n"
	_, err := LoadCatalog([]byte(src))
	if err == nil {
		t.Fatal("a source without a slug was accepted")
	}
	if !errors.Is(err, yaml.ErrRequired) {
		t.Errorf("not reported as incomplete: %v", err)
	}
	if !strings.Contains(err.Error(), "incomplete") ||
		!strings.Contains(err.Error(), "slug") {
		t.Errorf("the message does not say what is missing: %v", err)
	}
}

// A merge key supplies a value, so the key counts as present. Required
// and the decoder have to agree on that, or a catalog that reads fine
// would fail to load.
func TestMergeSatisfiesRequired(t *testing.T) {
	src := `
version: 1
defaults: &d
  slug: from-merge
sources:
  - <<: *d
    endpoint: https://example.com/x
`
	c, err := LoadCatalog([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].Slug != "from-merge" {
		t.Errorf("slug = %q", c.Sources[0].Slug)
	}
}

func TestTypoIsAnError(t *testing.T) {
	src := "version: 1\nsources:\n  - slug: x\n    endpiont: https://a/b\n"
	_, err := LoadCatalog([]byte(src))
	if err == nil {
		t.Fatal("a misspelled key was skipped")
	}
	if !strings.Contains(err.Error(), "endpiont") ||
		!strings.Contains(err.Error(), "line 4") {
		t.Errorf("the message does not point at the typo: %v", err)
	}
}

// A duration written as a bare number would be nanoseconds, which nobody
// ever means in a configuration file.
func TestBareNumberIsNotADuration(t *testing.T) {
	src := minimal + "    timeout: 30\n"
	_, err := LoadCatalog([]byte(src))
	if err == nil {
		t.Fatal("30 nanoseconds was accepted as a timeout")
	}
	if !strings.Contains(err.Error(), `"30s"`) {
		t.Errorf("the message does not say how to write it: %v", err)
	}
}

func TestExpansionFillsTokensAndLeavesMoneyAlone(t *testing.T) {
	t.Setenv("CATALOG_TOKEN", "s3cr3t")
	src := minimal + "    token: ${CATALOG_TOKEN}\n    note: \"costs $100\"\n"
	c, err := LoadCatalog([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].Token != "s3cr3t" {
		t.Errorf("token = %q", c.Sources[0].Token)
	}
	if c.Sources[0].Note != "costs $100" {
		t.Errorf("note = %q: a price is not a variable", c.Sources[0].Note)
	}
}

func TestUnsetVariableRefusesToStart(t *testing.T) {
	src := minimal + "    token: ${DEFINITELY_NOT_SET_HERE}\n"
	_, err := LoadCatalog([]byte(src))
	if err == nil {
		t.Fatal("a token that expands to nothing was accepted")
	}
	if !strings.Contains(err.Error(), "DEFINITELY_NOT_SET_HERE") {
		t.Errorf("the message does not name the variable: %v", err)
	}
}

func TestValidateCoversWhatATagCannot(t *testing.T) {
	src := `
version: 1
sources:
  - slug: same
    endpoint: https://a/b
  - slug: same
    endpoint: https://c/d
`
	_, err := LoadCatalog([]byte(src))
	if err == nil || !strings.Contains(err.Error(), "twice") {
		t.Fatalf("a duplicate slug was accepted: %v", err)
	}
}

// The uploaded format is not ours, so a key we do not know is a field we
// do not need yet.
func TestManifestKeepsUnknownKeys(t *testing.T) {
	m, err := LoadManifest([]byte(uploaded))
	if err != nil {
		t.Fatal(err)
	}
	if m.Title != "Field notes" || len(m.Pages) != 1 {
		t.Fatalf("got %+v", m)
	}
}

func TestManifestStillNeedsATitle(t *testing.T) {
	_, err := LoadManifest([]byte("pages:\n  - title: One\n"))
	if !errors.Is(err, yaml.ErrRequired) {
		t.Fatalf("a manifest without a title was accepted: %v", err)
	}
}

func TestExportRedactsAndRoundTrips(t *testing.T) {
	t.Setenv("CATALOG_TOKEN", "s3cr3t")
	c, err := LoadCatalog([]byte(curated))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Export(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "s3cr3t") {
		t.Fatalf("the export carries the token:\n%s", out)
	}
	if !strings.Contains(string(out), "timeout: 45s") {
		t.Errorf("a duration was not written as text:\n%s", out)
	}
	if !strings.Contains(string(out), "endpoint: https://example.com/feed.xml") {
		t.Errorf("a URL was not written as text:\n%s", out)
	}

	// The export is a catalog in its own right, minus the tokens.
	back, err := LoadCatalog(out)
	if err != nil {
		t.Fatalf("the export does not load back: %v", err)
	}
	if len(back.Sources) != len(c.Sources) {
		t.Errorf("round trip lost sources")
	}
}

func TestExportIsStable(t *testing.T) {
	t.Setenv("CATALOG_TOKEN", "s3cr3t")
	c, err := LoadCatalog([]byte(curated))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := Export(c)
	b, _ := Export(c)
	if string(a) != string(b) {
		t.Error("the same catalog encoded to different bytes")
	}
}
