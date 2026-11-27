package plex_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/internal/plextest"
)

func TestHTTPClientPINFlowAgainstPlexTV(t *testing.T) {
	t.Parallel()
	tv := plextest.New(t)
	client := plex.NewHTTPClient(tv.URL)
	ctx := context.Background()

	pin, err := client.CreatePIN(ctx, "client-1")
	if err != nil {
		t.Fatal(err)
	}
	if pin.ID == 0 || pin.Code == "" {
		t.Fatalf("pin = %+v", pin)
	}

	token, err := client.CheckPIN(ctx, "client-1", pin)
	if err != nil || token != "" {
		t.Fatalf("pending pin: token %q err %v", token, err)
	}

	tv.Approve(pin.Code, "tok-1", "alice")
	token, err = client.CheckPIN(ctx, "client-1", pin)
	if err != nil || token != "tok-1" {
		t.Fatalf("approved pin: token %q err %v", token, err)
	}

	name, err := client.Username(ctx, "client-1", "tok-1")
	if err != nil || name != "alice" {
		t.Fatalf("username %q err %v", name, err)
	}
}

func TestHTTPClientReportsRevokedToken(t *testing.T) {
	t.Parallel()
	tv := plextest.New(t)
	client := plex.NewHTTPClient(tv.URL)

	_, err := client.Username(context.Background(), "client-1", "never-issued")
	if !errors.Is(err, plex.ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestHTTPClientErrorsWhenPlexTVIsDown(t *testing.T) {
	t.Parallel()
	tv := plextest.New(t)
	tv.SetDown(true)
	client := plex.NewHTTPClient(tv.URL)

	_, err := client.Username(context.Background(), "client-1", "tok")
	if err == nil || errors.Is(err, plex.ErrUnauthorized) {
		t.Errorf("err = %v, want a non-unauthorized error", err)
	}
	if _, err := client.CreatePIN(context.Background(), "client-1"); err == nil {
		t.Error("CreatePIN should fail")
	}
}

func TestAuthURLCarriesClientIdentifierAndCode(t *testing.T) {
	t.Parallel()
	u := plex.AuthURL("client-1", plex.PIN{ID: 5, Code: "abcd"})
	for _, want := range []string{"https://app.plex.tv/auth", "clientID=client-1", "code=abcd"} {
		if !strings.Contains(u, want) {
			t.Errorf("%q missing %q", u, want)
		}
	}
}
