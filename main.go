package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/charmbracelet/log"

	"github.com/ReidMason/plex-ani-sync/internal/animeList"
	"github.com/ReidMason/plex-ani-sync/internal/api"
	"github.com/ReidMason/plex-ani-sync/internal/clock"
	"github.com/ReidMason/plex-ani-sync/internal/mapping"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/request"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	synchandler "github.com/ReidMason/plex-ani-sync/internal/syncHandler"
)

const dbLocation = "data/data.db"

type cmdArgs struct {
	listenAddr string
}

func run(w io.Writer, args cmdArgs) error {
	handler := log.New(w)
	handler.SetLevel(log.DebugLevel)
	logger := slog.New(handler)
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.SetDefault(logger)

	logger.Debug("Starting up", slog.String("listenAddr", args.listenAddr))

	logger.Info("Initialising storage")
	storage, err := storage.NewSqliteStorage(dbLocation, logger)
	if err != nil {
		logger.Error("Failed to initialise storage", slog.Any("error", err))
		return err
	}

	// err = storage.Reset()
	// if err != nil {
	// 	logger.Error("Failed to reset database", slog.Any("error", err))
	// }

	logger.Info("Applying migrations")
	err = storage.ApplyMigrations()
	if err != nil {
		logger.Error("Failed to apply migrations", slog.Any("error", err))
		return err
	}

	logger.Info("Initialising media host")
	plex := mediaHost.NewPlex()

	logger.Info("Initialising animeList service")
	client := request.NewStaggeredHttpClient(http.DefaultClient, logger)
	anilist := animeList.NewAnilist(client, storage, logger)

	logger.Info("Initialising mapping finder")
	mappingFinder := mapping.NewMappingFinder(anilist, logger)

	logger.Info("Initialising sync service")
	syncService := synchandler.NewSyncHandler(logger, clock.New())

	logger.Info("Initialising server")
	server := api.NewServer(args.listenAddr, syncService, mappingFinder, storage, plex, anilist, storage, logger)
	logger.Info("Initialising setting up media host")

	syncService = synchandler.NewSyncHandler(logger, clock.New())
	server.SyncService = syncService

	logger.Info("Starting server")
	server.Start()
	return nil
}

func main() {
	args := cmdArgs{
		listenAddr: *flag.String("listen-addr", ":8000", "server listen address"),
	}
	flag.Parse()

	if err := run(os.Stdout, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
