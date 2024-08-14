package output

import (
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
)

type scanner interface {
	Scan(dest ...interface{}) error
}

var scanColumns = []string{
	"sequence",
	"created_at",
	"hash",
	"`index`",
	"user_id",
	"app_id",
	"asset_id",
	"amount",
}

func scanOutput(scanner scanner, output *core.Output) error {
	var hash string

	if err := scanner.Scan(
		&output.Sequence,
		&output.CreatedAt,
		&hash,
		&output.Index,
		&output.UserID,
		&output.AppID,
		&output.AssetID,
		&output.Amount,
	); err != nil {
		return err
	}

	output.Hash = generic.Must(mixinnet.HashFromString(hash))
	return nil
}
