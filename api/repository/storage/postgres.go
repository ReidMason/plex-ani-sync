package storage

import "github.com/ReidMason/plex-ani-sync/internal/domain/plexAuth"

type Postgres struct{}

func NewPostgres() *Postgres {
	return &Postgres{}
}

func (p *Postgres) setToken(token plexAuth.PlexToken) error {
	return nil
}

func (p *Postgres) getPrivateKey() ([]byte, error) {
	return make([]byte, 32), nil
}
