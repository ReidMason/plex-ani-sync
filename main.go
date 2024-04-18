package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/api"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/charmbracelet/log"
)

type cmdArgs struct {
	listenAddr string
	dbUser     string
	dbPass     string
	dbHost     string
	dbPort     string
	dbName     string
}

func run(w io.Writer, args cmdArgs) error {
	handler := log.New(w)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	storage, err := storage.NewPostgresStorage(args.dbUser, args.dbPass, args.dbHost, args.dbPort, args.dbName)
	if err != nil {
		slog.Error("Failed to initialise storage", slog.Any("error", err))
		return err
	}

	plex := mediaHost.NewPlex()

	client := http.Client{}
	anilist := animeList.NewAnilist(&client, 392754)
	animeList, err := anilist.GetAnimeList()
	if err != nil {
		slog.Error("Failed to get anime list", slog.Any("error", err))
		return err
	}

	for _, entry := range animeList {
		log.Infof("Anime ID: %s, Status: %s", entry.AnimeId, entry.Status)
	}

	results, err := anilist.SearchAnime("Naruto")
	if err != nil {
		slog.Error("Failed to search anime", slog.Any("error", err))
		return err
	}

	slog.Info("Found anime", slog.Any("anime", results))

	server := api.NewServer(args.listenAddr, plex, storage)
	if err := server.Start(); err != nil {
		slog.Error("Failed to start server", slog.Any("error", err))
		return err
	}

	return nil
}

func main() {
	args := cmdArgs{
		listenAddr: *flag.String("listen-addr", ":8000", "server listen address"),
		dbUser:     *flag.String("db-user", "admin", "database user"),
		dbPass:     *flag.String("db-pass", "admin", "database password"),
		dbHost:     *flag.String("db-host", "localhost", "database host"),
		dbPort:     *flag.String("db-port", "5432", "database port"),
		dbName:     *flag.String("db-name", "plexanilistsync", "database name"),
	}
	flag.Parse()

	if err := run(os.Stdout, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
