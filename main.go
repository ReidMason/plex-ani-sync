package main

import (
	"flag"
	"log"

	"github.com/ReidMason/plex-ani-sync/api"
	"github.com/ReidMason/plex-ani-sync/internal/storage"
)

func main() {
	listenAddr := flag.String("listen-addr", ":8000", "server listen address")
	flag.Parse()

	storage, err := storage.NewPostgresStorage("testuser", "testpass", "localhost", "5432", "plexAnilistSync")
	if err != nil {
		log.Fatalf("Failed to initialise storage: %v", err)
	}

	server := api.NewServer(*listenAddr, storage)
	log.Fatal(server.Start())
}
