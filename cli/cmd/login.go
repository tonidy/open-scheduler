package cmd

import (
	"fmt"
	"os"

	"github.com/open-scheduler/cli/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Centro server",
	Long:  "Authenticate with Centro server and save JWT token",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		
		if username == "" {
			fmt.Print("Username: ")
			fmt.Scanln(&username)
		}
		
		if password == "" {
			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password = string(passwordBytes)
			fmt.Println()
		}
		
		if username == "" || password == "" {
			return fmt.Errorf("username and password are required")
		}
		
		c := client.NewClient(getBaseURL())
		if err := c.Login(username, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		
		fmt.Println("Login successful! Token saved.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
}

