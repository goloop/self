// Recipe 012: the refresh-token lifecycle, done safely.
//
// An access token is short-lived on purpose; a refresh token is how a client
// gets the next one without asking the user to sign in again. That makes the
// refresh token the valuable thing to steal, and the whole lifecycle is built
// around one question: when a token that is no longer current comes back, is
// this an honest client repeating a request whose answer it never saw, or an
// attacker replaying a token they took?
//
// goloop/auth answers it with rotation. Every refresh issues a new token and
// retires the old one, and presenting a retired token is a signal:
//
//	A. rotate      - RotateWithStatus issues the successor and reports Rotated;
//	B. reuse       - a token already rotated comes back as ReusedStale, and the
//	                 response is to revoke the whole family (RevokeAll);
//	C. grace       - the token rotated a moment ago, replayed within a short
//	                 window, is PreviousWithinGrace: a dropped response, not a
//	                 theft, so answer 401 without punishing the family;
//	D. expiry      - a token past its lifetime is ErrRefreshExpired, which is
//	                 not reuse: an idle client is not a thief;
//	E. sign out    - RevokeAll ends every session a subject has.
//
// The recipe uses auth.NewMemoryRefreshStore so it runs with zero
// infrastructure. A production service puts this behind Redis or Postgres; the
// contract is identical, and auth/authtest (see main_test.go) proves any
// implementation obeys it. The chapter shows the atomic Redis rotation.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/goloop/auth"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// This store has no grace window, so a retired token replayed is
	// unambiguously reuse - the theft signal. Example C opens a window and
	// shows the softer reading.
	store := auth.NewMemoryRefreshStore()

	// issue mints a token for a subject and saves the record. In a service
	// this happens at sign-in and returns the opaque token to the client.
	issue := func(subject string) auth.RefreshToken {
		rt, _, _ := auth.NewRefreshToken(subject, time.Hour)
		_ = store.Save(ctx, rt)
		return rt
	}

	// rotate is the whole flow a /refresh endpoint runs: mint the successor,
	// swap it for the presented token, and act on the status.
	rotate := func(oldID string) (auth.RotateResult, auth.RefreshToken, error) {
		next, _, _ := auth.NewRefreshToken("ada", time.Hour)
		res, err := auth.RotateWithStatus(ctx, store, oldID, next)
		return res, next, err
	}

	fmt.Println("A. a normal refresh issues the next token:")
	first := issue("ada")
	res, second, err := rotate(first.ID)
	if err != nil {
		return err
	}
	fmt.Printf("   status=%s  a successor was issued\n", res.Status)

	fmt.Println("B. replaying the retired token is a theft signal:")
	// A real /refresh handler loads the record and verifies the secret
	// before it rotates, so it already knows the subject from that step. We
	// take the same subject the sign-in knew ("ada"): RotateResult.Subject
	// is filled where the store can still say, but a stale token whose
	// record is long gone cannot always name its owner, so hold onto it
	// yourself rather than depend on the result.
	const owner = "ada"
	res, _, err = rotate(first.ID) // the old token, again
	fmt.Printf("   status=%s  err=%v\n", res.Status, err)
	if res.Status == auth.ReusedStale {
		// The response to reuse: end every session this subject has. The
		// stolen token and the legitimate one are now both dead, and the
		// user signs in again - which is the correct outcome of a theft.
		if err := auth.RevokeAll(ctx, store, owner); err != nil {
			return err
		}
		fmt.Printf("   -> RevokeAll(%q): the whole family is revoked\n", owner)
	}

	fmt.Println("C. a grace window absorbs a dropped response:")
	graceStore := auth.NewMemoryRefreshStore(auth.WithGrace(time.Minute))
	g1, _, _ := auth.NewRefreshToken("bob", time.Hour)
	_ = graceStore.Save(ctx, g1)
	g2, _, _ := auth.NewRefreshToken("bob", time.Hour)
	_, _ = auth.RotateWithStatus(ctx, graceStore, g1.ID, g2)
	// The client never saw g2 (its connection dropped) and retries g1.
	res, _ = auth.RotateWithStatus(ctx, graceStore, g1.ID,
		mustToken("bob"))
	fmt.Printf("   replaying the just-rotated token: status=%s\n", res.Status)
	fmt.Println("   -> answer 401, but do NOT revoke: this is a lost response, not theft")

	fmt.Println("D. an expired token is refused, but is not reuse:")
	expStore := auth.NewMemoryRefreshStore()
	shortLived, _, _ := auth.NewRefreshToken("carol", time.Millisecond)
	_ = expStore.Save(ctx, shortLived)
	time.Sleep(5 * time.Millisecond)
	_, err = auth.RotateWithStatus(ctx, expStore, shortLived.ID, mustToken("carol"))
	fmt.Printf("   err is ErrRefreshExpired: %v (and NOT ErrRefreshUsed: %v)\n",
		errors.Is(err, auth.ErrRefreshExpired), errors.Is(err, auth.ErrRefreshUsed))

	fmt.Println("E. sign out everywhere:")
	signOutStore := auth.NewMemoryRefreshStore()
	for i := 0; i < 3; i++ {
		t, _, _ := auth.NewRefreshToken("dave", time.Hour)
		_ = signOutStore.Save(ctx, t)
	}
	if err := auth.RevokeAll(ctx, signOutStore, "dave"); err != nil {
		return err
	}
	fmt.Println("   -> every one of dave's refresh tokens is now revoked")

	_ = second // the successor from A, unused past the demo
	return nil
}

// mustToken mints a token or panics; the recipe only uses it where an error is
// impossible (a fixed subject and ttl), so it keeps the narrative uncluttered.
func mustToken(subject string) auth.RefreshToken {
	rt, _, err := auth.NewRefreshToken(subject, time.Hour)
	if err != nil {
		panic(err)
	}
	return rt
}
