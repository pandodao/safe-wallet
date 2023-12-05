package db

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed schema/*.sql
var fs embed.FS

// Migrate run mysql migration with embed schemes.
func Migrate(db *sql.DB) error {
	d, err := iofs.New(fs, "schema")
	if err != nil {
		return err
	}

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", d, "mysql", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
