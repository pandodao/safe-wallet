package cmds

import (
	"context"
	"encoding/json"
	"os"

	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
	"github.com/spf13/cobra"
)

type Cmd struct {
	Wallets core.WalletStore
}

func (c *Cmd) Run(ctx context.Context, args []string) error {
	root := &cobra.Command{
		Use:   "safe-wallet",
		Short: "safe-wallet",
	}

	root.AddCommand(c.exportAllWalletsCmd())
	root.AddCommand(c.exportWalletCmd())

	root.SetArgs(args)
	root.SetOut(os.Stdout)

	return root.ExecuteContext(ctx)
}

func (c *Cmd) exportAllWalletsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export-wallets",
		Short: "export all subwallets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			wallets, err := c.Wallets.List(ctx)
			if err != nil {
				return err
			}

			return jsonPrint(cmd, generic.MapSlice(wallets, keystoreFromWallet))
		},
	}
}

func (c *Cmd) exportWalletCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export-wallet",
		Short: "export a subwallet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			userID := args[0]
			wallet, err := c.Wallets.Find(ctx, userID)
			if err != nil {
				return err
			}

			return jsonPrint(cmd, keystoreFromWallet(wallet))
		},
	}
}

func jsonPrint(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
