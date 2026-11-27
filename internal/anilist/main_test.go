package anilist_test

import (
	"os"
	"testing"

	"github.com/ReidMason/plex-ani-sync/internal/loopbacktest"
)

func TestMain(m *testing.M) { os.Exit(loopbacktest.Run(m)) }
