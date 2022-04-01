package main

import (
	"net/http"

	"github.com/lbajolet/atm_service/pkg/api"
	"github.com/lbajolet/atm_service/pkg/persistence"
	"github.com/spf13/cobra"
)

var rootCmd = cobra.Command{
	RunE: doMain,
	Use:  "atm: run the ATM service",
}

func main() {
	rootCmd.Execute()
}

func doMain(cmd *cobra.Command, args []string) error {
	db, err := persistence.NewDB()
	if err != nil {
		return err
	}

	srv := api.NewServer(db)
	return http.ListenAndServe("0.0.0.0:8080", srv)
}
