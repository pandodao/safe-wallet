/*
Copyright © 2024 pando
*/
package cmd

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/pandodao/safe-wallet/handler/rpc/safewallet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "safewallet-cli",
	Short: "rpc cmd for safe-wallet service",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("endpoint", "l", "http://localhost:8080", "rpc endpoint")
	viper.BindPFlag("endpoint", rootCmd.PersistentFlags().Lookup("endpoint"))
}

func getTwirpClient() safewallet.SafeWalletService {
	return safewallet.NewSafeWalletServiceProtobufClient(viper.GetString("endpoint"), http.DefaultClient)
}

func printJson(cmd *cobra.Command, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	cmd.Println(string(b))
	return nil
}
