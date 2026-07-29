package commands

import (
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
	RunE:  runWeb,
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
	ResolveCommand(buildFlagArgs())
}
