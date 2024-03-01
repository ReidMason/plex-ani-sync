package main

import (
	"flag"
	"log"

	"github.com/ReidMason/plex-ani-sync/api"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
	plexAnilistSyncDb "github.com/ReidMason/plex-ani-sync/internal/storage/postgres"
)

func main() {
	listenAddr := flag.String("listen-addr", ":8000", "server listen address")
	flag.Parse()

	storage := initStorage()
	server := api.NewServer(*listenAddr, storage)
	log.Fatal(server.Start())
}

func initStorage() storage.Postgres {
	connectionString := storage.BuildConnectionString("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	driver, err := storage.ConnectToDatabase(connectionString)
	if err != nil {
		panic(err)
	}

	queries := plexAnilistSyncDb.New(driver)
	return storage.NewPostgresStorage(queries)
}
