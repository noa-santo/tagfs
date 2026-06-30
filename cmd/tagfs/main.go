package main

import (
	"fmt"
	"os"

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
			if storage == "" || mount == "" {
				fmt.Println("Error: --storage and --mount are required")
				os.Exit(1)
			}
			fmt.Println("Starting tagfs Daemon...")
			fuse.StartDaemon(storage, mount)
		},
	}
	mountCmd.Flags().StringVar(&storage, "storage", "", "Path to physical storage")
	mountCmd.Flags().StringVar(&mount, "mount", "", "Path to mount point")

	var unmountCmd = &cobra.Command{
		Use:   "unmount",
		Short: "Unmounts the tagfs FUSE daemon",
		Run: func(cmd *cobra.Command, args []string) {
			// todo
		},
	}

	var manageCmd = &cobra.Command{
		Use:   "manage",
		Short: "Starts the management TUI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Opening Inbox Manager...")
			// todo: Call your Bubbletea TUI initialization logic here
		},
	}

	rootCmd.AddCommand(mountCmd, unmountCmd, manageCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
