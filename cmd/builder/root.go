package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "builder",
	Short: "Photography Portfolio Builder CLI",
	Long:  `A CLI tool to process images and generate a static photography portfolio website.`,
}

var envFile string

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be defined here
	rootCmd.PersistentFlags().StringVar(&envFile, "env", "", "Path to .env file to load before running commands")

	// Load .env file if provided before any command runs
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if envFile == "" {
			return nil
		}
		if err := godotenv.Load(envFile); err != nil {
			return fmt.Errorf("failed to load env file '%s': %w", envFile, err)
		}
		return nil
	}
}
