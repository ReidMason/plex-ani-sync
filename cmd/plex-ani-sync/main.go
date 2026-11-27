package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/config"
	"github.com/ReidMason/plex-ani-sync/internal/server"
	"github.com/ReidMason/plex-ani-sync/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	addr := ":8080"
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		addr = v
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	handler, cleanup, err := server.New(ctx, server.Config{
		Env:           cfg,
		Assets:        web.Assets(),
		PlexTVURL:     "https://plex.tv",
		FribbURL:      "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json",
		AnimeListsURL: "https://raw.githubusercontent.com/Anime-Lists/anime-lists/master/anime-list.xml",
		AniListURL:    "https://graphql.anilist.co",
	})
	if err != nil {
		return err
	}
	defer cleanup()

	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
