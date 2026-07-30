package cli

import (
	"fmt"
	"os"

	"git.itopcms.com/jackliu/doc/internal/app"
	"github.com/spf13/cobra"
)

var (
	configFile string
	workingDir string
	logFile    string
)

var rootCmd = &cobra.Command{
	Use:   "doc",
	Short: "Doc — Documentation & knowledge base server",
	Long:  "Doc is a documentation management server. Run without a subcommand to start the web service.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		preflightCheck()
	},
	RunE: runWeb,
}

// Execute is the cobra entry point used by main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "configuration file path")
	rootCmd.PersistentFlags().StringVar(&workingDir, "dir", "", "working directory (overrides DOC_HOME)")
	rootCmd.PersistentFlags().StringVar(&logFile, "log", "", "log file path")
}

// buildFlagArgs reconstructs flag args for Daemon.ResolveCommand compatibility.
func buildFlagArgs() []string {
	var args []string
	if configFile != "" {
		args = append(args, "--config", configFile)
	}
	if workingDir != "" {
		args = append(args, "--dir", workingDir)
	}
	if logFile != "" {
		args = append(args, "--log", logFile)
	}
	return args
}

// bootstrapFromFlags initializes config/DB/cache using cobra persistent flags.
func bootstrapFromFlags() {
	app.ResolveCommand(buildFlagArgs())
}

// preflightCheck warns about Round 1/legacy layout leftovers. Does not abort.
func preflightCheck() {
	if _, err := os.Stat("./configs"); err == nil {
		fmt.Println("[warn] detected legacy ./configs; please migrate to ./conf (Beego default path)")
	}
	if _, err := os.Stat("./static"); err == nil {
		fmt.Println("[warn] detected legacy ./static; please migrate to ./web/static")
	}
	if _, err := os.Stat("./views"); err == nil {
		fmt.Println("[warn] detected legacy ./views; please migrate to ./web/views")
	}
}
