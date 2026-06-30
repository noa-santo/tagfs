package main

import (
	"fmt"
	"os"

	"github.com/noa-santo/tagfs/internal/db"
	"github.com/noa-santo/tagfs/internal/fuse"
	"github.com/spf13/cobra"
)

func main() {
	var storage, mount string
	var rootCmd = &cobra.Command{Use: "tagfs"}

	var mountCmd = &cobra.Command{
		Use:   "mount",
		Short: "Starts the tagfs FUSE daemon",
		Run: func(cmd *cobra.Command, args []string) {
			var config db.Config

			if storage != "" && mount != "" {
				config = db.Config{MountPath: mount, StoragePath: storage}
				if err := db.SaveConfig(config); err != nil {
					fmt.Printf("Failed to save config: %v\n", err)
				}
			} else {
				var err error
				config, err = db.LoadConfig()
				if err != nil {
					fmt.Println("No configuration found. Please run with --storage and --mount.")
					os.Exit(1)
				}
			}
			fmt.Println("Starting tagfs Daemon...")
			fuse.StartDaemon(config.StoragePath, config.MountPath)
		},
	}
	mountCmd.Flags().StringVar(&storage, "storage", "", "Path to physical storage")
	mountCmd.Flags().StringVar(&mount, "mount", "", "Path to mount point")

	var manageCmd = &cobra.Command{
		Use:   "manage",
		Short: "Starts the management TUI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Opening Inbox Manager...")
			// todo: Call your Bubbletea TUI initialization logic here
		},
	}

	rootCmd.AddCommand(mountCmd, manageCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
