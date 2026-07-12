package main

import (
	"log"
	"os"

	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/fuse"
	"github.com/noa-santo/tagfs/internal/tui"
	"github.com/spf13/cobra"
)

var mainLogger = log.New(os.Stdout, "MAIN: ", log.LstdFlags)

func main() {
	var configPath string
	var rootCmd = &cobra.Command{Use: "tagfs"}

	var mountCmd = &cobra.Command{
		Use:   "mount",
		Short: "Starts the tagfs FUSE daemon",
		Run: func(cmd *cobra.Command, args []string) {
			if configPath == "" {
				mainLogger.Panic("No config supplied! Supply a config with --config <path>")
			}
			err := config.InitConfig(configPath)
			if err != nil {
				mainLogger.Panicf("Error while loading config: %v", err)
			}
			mainLogger.Println("Starting tagfs Daemon...")
			fuse.StartDaemon()
		},
	}
	mountCmd.Flags().StringVar(&configPath, "config", "", "Path to config file")

	var manageCmd = &cobra.Command{
		Use:   "inbox",
		Short: "Starts the inbox management TUI",
		Run: func(cmd *cobra.Command, args []string) {
			mainLogger.Println("Opening Inbox Manager...")
			tui.StartTui()
		},
	}

	rootCmd.AddCommand(mountCmd, manageCmd)

	if err := rootCmd.Execute(); err != nil {
		mainLogger.Fatalf("Error executing command: %v\n", err)
	}
}
