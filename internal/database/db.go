package database

import (
	"database/sql"
	"embed"
	_ "embed"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func New(path string) (*sql.DB, error) {
	// The connection string uses a file: URI
	// and the _pragma parameter to set journal_mode to WAL.
	connStr := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, err
	}
	// setup database

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite"); err != nil {
		panic(err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		panic(err)
	}

	return db, nil
}
