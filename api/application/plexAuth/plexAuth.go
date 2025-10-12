package plexAuth

import "github.com/ReidMason/plex-ani-sync/internal/domain/common"

type PlexAuth struct {
	PlexAuthApiRegistry PlexAuthApiRepository
	StorageRepository   StorageRepository
	time                common.TimeRepository
}

func New(plexAuthApiRegistry PlexAuthApiRepository, storageRepository StorageRepository, time common.TimeRepository) *PlexAuth {
	return &PlexAuth{
		PlexAuthApiRegistry: plexAuthApiRegistry,
		StorageRepository:   storageRepository,
		time:                time,
	}
}
