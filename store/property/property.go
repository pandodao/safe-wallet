package property

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pandodao/safe-wallet/core"
	"github.com/tsenart/nap"
)

type store struct {
	db *nap.DB
}

func New(db *nap.DB) core.PropertyStore {
	return &store{db: db}
}

func (s *store) Get(ctx context.Context, key string, value any) error {
	var raw []byte
	if err := s.db.QueryRowContext(ctx, "SELECT `value` FROM properties WHERE `key` = ?", key).Scan(&raw); err == nil {
		return json.Unmarshal(raw, value)
	} else if errors.Is(err, sql.ErrNoRows) {
		return nil
	} else {
		return err
	}
}

func (s *store) Set(ctx context.Context, key string, value any) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	r, err := s.db.ExecContext(ctx, "UPDATE `properties` SET `value` = ?, `version` = `version` + 1 WHERE `key` = ?", jsonValue, key)
	if err != nil {
		return fmt.Errorf("failed to set property: %w", err)
	}

	n, err := r.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if n > 0 {
		return nil
	}

	_, err = s.db.ExecContext(ctx, "INSERT INTO `properties` (`key`, `value`) VALUES (?, ?)", key, jsonValue)
	return err

}
