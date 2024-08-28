package db

import (
	"bytes"
	"database/sql"
	"embed"
	"errors"
	"html/template"
	"io"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed schema/*.sql
var embedFiles embed.FS

type MigrateData struct {
	UserID string
}

// Migrate run mysql migration with embed schemes.
func Migrate(db *sql.DB, data MigrateData) error {
	d, err := iofs.New(&templateFS{
		data: data,
		FS:   embedFiles,
	}, "schema")
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

type templateFile struct {
	io.ReadCloser
	info *fileInfoWithSize
}

func (t *templateFile) Stat() (fs.FileInfo, error) {
	return t.info, nil
}

type templateFS struct {
	data any
	embed.FS
}

func (t *templateFS) Open(name string) (fs.File, error) {
	file, err := t.FS.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Read the original file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Parse the content as a template
	tmpl, err := template.New("sql").Parse(string(content))
	if err != nil {
		return nil, err
	}

	// Execute the template with the provided data
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, t.data); err != nil {
		return nil, err
	}

	return &templateFile{
		ReadCloser: io.NopCloser(bytes.NewReader(buf.Bytes())),
		info:       &fileInfoWithSize{info, int64(buf.Len())},
	}, nil
}

type fileInfoWithSize struct {
	fs.FileInfo
	size int64
}

func (f *fileInfoWithSize) Size() int64 {
	return f.size
}
