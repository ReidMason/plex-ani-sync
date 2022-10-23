package main

import (
	"log"
	"plex-ani-sync/controllers"
	"plex-ani-sync/services/anilist"
	"plex-ani-sync/services/config"
	"plex-ani-sync/services/plex"
	"plex-ani-sync/services/requesthandler"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	configService := config.NewConfigHandler()
	requesthandler := requesthandler.New()
	plexService := plex.New(configService, requesthandler)
	anilistService := anilist.New(requesthandler)

	router.GET("/plex/libraries", controllers.GetLibraries(plexService))
	router.GET("/plex/libraries/:libraryId/series", controllers.GetSeries(plexService))

	router.GET("/anilist/setup/auth-confirmation", controllers.AuthConfirmation(configService))
	router.GET("/anilist/setup/auth-confirmation", func(ctx *gin.Context) {
		anilistService.GetAccessToken()
	})

	clientId := "6512"
	redirectUri := "http://10.128.0.160:5050/anilist/setup/auth-confirmation"
	url := "https://anilist.co/api/v2/oauth/authorize?client_id=" + clientId + "&redirect_uri=" + redirectUri + "&response_type=code"

	log.Print(url)

	err := router.Run(":5050")
	if err != nil {
		log.Fatal("Failed to start web server")
	}
}
