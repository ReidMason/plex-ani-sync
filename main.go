package main

import (
	"flag"
	"log"

	"github.com/ReidMason/plex-ani-sync/api"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

func main() {
	listenAddr := flag.String("listen-addr", ":8000", "server listen address")
	dbUser := flag.String("db-user", "admin", "database user")
	dbPass := flag.String("db-pass", "admin", "database password")
	dbHost := flag.String("db-host", "localhost", "database host")
	dbPort := flag.String("db-port", "5432", "database port")
	dbName := flag.String("db-name", "plexanilistsync", "database name")
	flag.Parse()

	storage, err := storage.NewPostgresStorage(*dbUser, *dbPass, *dbHost, *dbPort, *dbName)
	if err != nil {
		log.Fatalf("Failed to initialise storage: %v", err)
		panic(err)
	}

	server := api.NewServer(*listenAddr, storage)
	log.Fatal(server.Start())
}
