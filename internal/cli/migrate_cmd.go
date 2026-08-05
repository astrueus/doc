package cli

import (
	"git.itopcms.com/astrueus/doc/internal/migrate"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		bootstrapFromFlags()
		migrate.RegisterMigration()
		migrate.RunMigration()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
