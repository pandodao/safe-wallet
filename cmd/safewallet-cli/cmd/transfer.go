/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/google/uuid"
	"github.com/pandodao/safe-wallet/handler/rpc/safewallet"
	"github.com/spf13/cobra"
)

var transferOpt safewallet.CreateTransferRequest

// transferCmd represents the transfer command
var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "create a transfer by safewallet rpc",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return showTransfer(cmd, args[0])
		}

		// create transfer
		if transferOpt.TraceId == "" {
			transferOpt.TraceId = uuid.NewString()
		}

		if len(transferOpt.Opponents) == 1 {
			transferOpt.Threshold = 1
		}

		return createTransfer(cmd, &transferOpt)
	},
}

func init() {
	rootCmd.AddCommand(transferCmd)

	transferCmd.Flags().StringVar(&transferOpt.TraceId, "trace", "", "trace id (optional)")
	transferCmd.Flags().StringVar(&transferOpt.AssetId, "asset", "", "asset id")
	transferCmd.Flags().StringVar(&transferOpt.Amount, "amount", "0", "amount")
	transferCmd.Flags().StringVar(&transferOpt.Memo, "memo", "", "memo (optional)")
	transferCmd.Flags().StringSliceVar(&transferOpt.Opponents, "opponents", nil, "opponents")
	transferCmd.Flags().Uint32Var(&transferOpt.Threshold, "threshold", 0, "threshold")
}

func showTransfer(cmd *cobra.Command, id string) error {
	cmd.Println("show transfer:", id)
	resp, err := getTwirpClient().FindTransfer(cmd.Context(), &safewallet.FindTransferRequest{
		TraceId: id,
	})

	if err != nil {
		return err
	}

	return printJson(cmd, resp.Transfer)
}

func createTransfer(cmd *cobra.Command, req *safewallet.CreateTransferRequest) error {
	resp, err := getTwirpClient().CreateTransfer(cmd.Context(), req)

	if err != nil {
		return err
	}

	return printJson(cmd, resp.Transfer)
}
