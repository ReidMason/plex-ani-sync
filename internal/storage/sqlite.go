package storage

import (
	"database/sql"
	"embed"
	"log/slog"
	"time"

	sqlite3Storage "github.com/ReidMason/plex-ani-sync/internal/storage/sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Sqlite struct {
	db      *sql.DB
	queries *sqlite3Storage.Queries
	log     *slog.Logger
}

func NewSqliteStorage(databasePath string, logger *slog.Logger) (*Sqlite, error) {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return nil, err
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, err
	}

	return &Sqlite{
		db:      db,
		queries: sqlite3Storage.New(db),
		log:     logger,
	}, nil
}

func (s Sqlite) Reset() error {
	slog.Warn("Resetting database")
	return goose.Down(s.db, "migrations")
}

func (s Sqlite) ApplyMigrations() error {
	slog.Info("Applying migrations")
	return goose.Up(s.db, "migrations")
}

func parseIso8601Time(timeString string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeString)
}
