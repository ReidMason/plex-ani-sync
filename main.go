package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/ReidMason/plex-ani-sync/api"
	"github.com/ReidMason/plex-ani-sync/internal/mediaHost"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	"github.com/ReidMason/plex-ani-sync/internal/userManager"
	"github.com/charmbracelet/log"
)

func main() {
	listenAddr := flag.String("listen-addr", ":8000", "server listen address")
	dbUser := flag.String("db-user", "admin", "database user")
	dbPass := flag.String("db-pass", "admin", "database password")
	dbHost := flag.String("db-host", "localhost", "database host")
	dbPort := flag.String("db-port", "5432", "database port")
	dbName := flag.String("db-name", "plexanilistsync", "database name")
	flag.Parse()

	handler := log.New(os.Stdout)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	storage, err := storage.NewPostgresStorage(*dbUser, *dbPass, *dbHost, *dbPort, *dbName)
	userManagerService := userManager.NewUserManager(storage)
	if err != nil {
		log.Fatalf("Failed to initialise storage: %v", err)
		panic(err)
	}

	plex := mediaHost.NewPlex()

	server := api.NewServer(*listenAddr, plex, userManagerService)
	log.Fatal(server.Start())
}
