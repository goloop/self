// Recipe 007: readable URLs and identifiers from any text.
//
// The task: turn a human title into a clean URL slug (even when the title is in
// Cyrillic), keep it unique, and convert a name between naming styles. Three
// small modules:
//
//	A. slug  - a lower-case, unique URL slug;
//	B. t13n  - transliterate non-Latin text, then slug it;
//	C. scs   - convert one name between camel, snake, kebab, Pascal and more.
package main

import (
	"fmt"
	"os"

	"github.com/goloop/scs/v2"
	"github.com/goloop/slug/v2"
	"github.com/goloop/t13n/v2"
)

func main() {
	// Example A: a unique lower-case slug. MakeUnique appends -2, -3, ... while
	// an "already taken" function reports a clash.
	fmt.Println("A. unique URL slug (slug):")
	s := slug.New(slug.WithLowercase())
	taken := map[string]bool{"getting-started": true}
	for _, title := range []string{"Getting Started", "Getting Started", "Hello, World!"} {
		out := s.MakeUnique(title, func(x string) bool { return taken[x] })
		taken[out] = true
		fmt.Printf("   %-18q -> %q\n", title, out)
	}

	// Example B: transliterate, then slug. t13n turns Cyrillic into Latin; slug
	// then makes a clean URL segment.
	fmt.Println("B. transliterate then slug (t13n + slug):")
	tr := t13n.New()
	for _, title := range []string{"Привіт, світ", "Огляд архітектури"} {
		latin := tr.Make(title)
		fmt.Printf("   %-20q -> %-22q -> %q\n", title, latin, s.Make(latin))
	}

	// Example C: convert between naming styles with one Caser.
	fmt.Println("C. naming styles (scs):")
	c := scs.New()
	name := "userAPIToken"
	fmt.Printf("   from %q:\n", name)
	fmt.Printf("     snake=%q kebab=%q\n", c.ToSnake(name), c.ToKebab(name))
	fmt.Printf("     pascal=%q camel=%q\n", c.ToPascal(name), c.ToCamel(name))
	fmt.Printf("     screaming=%q title=%q\n", c.ToScreamingSnake(name), c.ToTitle(name))
	os.Exit(0)
}
