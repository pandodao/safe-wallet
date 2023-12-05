package store

import (
	"database/sql"
	"errors"
)

func IsErrNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
