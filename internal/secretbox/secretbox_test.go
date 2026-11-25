package secretbox_test

import (
	"bytes"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/secretbox"
)

func key(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func mustBox(t *testing.T, k []byte) *secretbox.Box {
	t.Helper()
	box, err := secretbox.New(k)
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestRoundTrip(t *testing.T) {
	box := mustBox(t, key(1))
	sealed, err := box.Seal([]byte("plex-token"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("plex-token")) {
		t.Fatal("ciphertext contains plaintext")
	}
	got, err := box.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "plex-token" {
		t.Errorf("got %q", got)
	}
}

func TestSealingTwiceGivesDifferentCiphertext(t *testing.T) {
	box := mustBox(t, key(1))
	a, _ := box.Seal([]byte("x"))
	b, _ := box.Seal([]byte("x"))
	if bytes.Equal(a, b) {
		t.Error("nonce reused")
	}
}

func TestOpenWithWrongKeyFails(t *testing.T) {
	sealed, _ := mustBox(t, key(1)).Seal([]byte("plex-token"))
	if _, err := mustBox(t, key(2)).Open(sealed); err == nil {
		t.Error("expected error")
	}
}

func TestOpenDetectsTampering(t *testing.T) {
	box := mustBox(t, key(1))
	sealed, _ := box.Seal([]byte("plex-token"))
	for i := range sealed {
		tampered := bytes.Clone(sealed)
		tampered[i] ^= 1
		if _, err := box.Open(tampered); err == nil {
			t.Fatalf("flipping byte %d went undetected", i)
		}
	}
	if _, err := box.Open(sealed[:5]); err == nil {
		t.Error("truncated input accepted")
	}
}

func TestNewRejectsWrongKeyLength(t *testing.T) {
	if _, err := secretbox.New([]byte("short")); err == nil {
		t.Error("expected error")
	}
}
