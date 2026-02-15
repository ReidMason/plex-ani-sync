package main

import (
	"fmt"

	"github.com/ReidMason/plex-ani-sync/internal/service"
)

func main() {
	fmt.Println("Hello, World!")

	anime := service.GetAnime()

	fmt.Println(anime)
}
