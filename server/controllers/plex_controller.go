package controllers

import (
	"net/http"
	"plex-ani-sync/services/plex"

	"github.com/gin-gonic/gin"
)

func GetLibraries(plexService plex.IPlexConnection) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		libraries, err := plexService.GetAllLibraries()

		if err != nil {
			ctx.IndentedJSON(http.StatusInternalServerError, err)
		}

		ctx.IndentedJSON(http.StatusOK, libraries)
	}
}

func GetSeries(plexService plex.IPlexConnection) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		libraryId := ctx.Param("libraryId")
		series, err := plexService.GetSeries(libraryId)

		if err != nil {
			ctx.IndentedJSON(http.StatusBadRequest, nil)
		}

		ctx.IndentedJSON(http.StatusOK, series)
	}
}
