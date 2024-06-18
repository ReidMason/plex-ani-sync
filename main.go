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
	logger := slog.New(handler)
	slog.SetDefault(logger)

	storage, err := storage.NewSqliteStorage(dbLocation, logger)
	if err != nil {
		logger.Error("Failed to initialise storage", slog.Any("error", err))
		return err
	}

	// err = storage.Reset()
	// if err != nil {
	// 	logger.Error("Failed to reset database", slog.Any("error", err))
	// }

	err = storage.ApplyMigrations()
	if err != nil {
		logger.Error("Failed to apply migrations", slog.Any("error", err))
		return err
	}

	plex := mediaHost.NewPlex()

	client := request.NewStaggeredHttpClient(http.DefaultClient, logger)
	anilist := animeList.NewAnilist(client, storage, logger)

	server := api.NewServer(args.listenAddr, plex, anilist, storage, logger)
	mediaHostService, err := server.InitialiseMediaHost()
	if err != nil {
		logger.Error("Failed to initialise media host", slog.Any("error", err))
	}

	mappingFinder := mapping.NewMappingFinder(anilist, storage, mediaHostService, logger)
	allSeries, err := mediaHostService.GetSeries("1")
	if err != nil {
		logger.Error("Failed to get series", slog.Any("error", err))
	}
	updateMappings(allSeries, mappingFinder, logger)

	syncHandler := synchandler.NewSyncHandler(mediaHostService, storage, mappingFinder, anilist, logger)
	syncHandler.Sync()

	server.Start()
	return nil
}

func updateMappings(allSeries []mediaHost.Series, mappingFinder mapping.MappingFinder, log *slog.Logger) {
	log.Info("Updating mappings")
	for _, series := range allSeries {
		log.Info("Updating mappings", slog.String("title", series.Title))
		err := mappingFinder.CreateMappings(series)
		if err != nil {
			log.Error("Failed to update mappings", slog.Any("error", err))
		}
	}
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
