package controllers

import (
	"log"
	"net/http"
	"plex-ani-sync/services/config"

	"github.com/gin-gonic/gin"
)

func AuthConfirmation(configService config.IConfigHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query := ctx.Request.URL.Query()
		codeQuery := query["code"]

		code := ""
		if len(codeQuery) > 0 {
			code = codeQuery[0]
		}

		log.Printf("Anilist auth code: %s", code)
		ctx.Redirect(http.StatusFound, "/plex/libraries")
	}
}
