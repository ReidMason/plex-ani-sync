package oidctest_test

import (
	"context"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/ReidMason/plex-ani-sync/internal/oidctest"
)

func TestProviderIsDiscoverable(t *testing.T) {
	t.Parallel()
	p := oidctest.New(t)

	provider, err := oidc.NewProvider(context.Background(), p.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := provider.Endpoint().AuthURL, p.URL+"/authorize"; got != want {
		t.Errorf("AuthURL = %q, want %q", got, want)
	}
}
