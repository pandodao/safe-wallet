/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/pandodao/safe-wallet/handler/rpc/safewallet"
	"github.com/spf13/cobra"
)

var walletOpt struct {
	safewallet.CreateWalletRequest
}

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "create a wallet",
	RunE: func(cmd *cobra.Command, args []string) error {
		return createWallet(cmd, &walletOpt.CreateWalletRequest)
	},
}

func init() {
	rootCmd.AddCommand(walletCmd)

	walletCmd.Flags().StringVar(&walletOpt.Label, "label", "", "label")
}

func createWallet(cmd *cobra.Command, req *safewallet.CreateWalletRequest) error {
	resp, err := getTwirpClient().CreateWallet(cmd.Context(), req)
	if err != nil {
		return err
	}

	return printJson(cmd, resp)
}
