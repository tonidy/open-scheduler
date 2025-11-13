package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	baseURL string
)

var rootCmd = &cobra.Command{
	Use:   "osctl",
	Short: "Open Scheduler CLI",
	Long:  "A CLI tool for managing Open Scheduler jobs, nodes, and instances",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&baseURL, "server", "", "Centro server URL (default: http://localhost:8080/api/v1)")
	
	// Set default base URL from environment variable
	if envURL := os.Getenv("OSCTL_SERVER"); envURL != "" {
		baseURL = envURL
	}
}

func getBaseURL() string {
	if baseURL != "" {
		return baseURL
	}
	return "http://localhost:8080/api/v1"
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

