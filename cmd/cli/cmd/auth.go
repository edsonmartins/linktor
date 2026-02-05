package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/linktor/msgfy/internal/client"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long: `Manage authentication with the Linktor platform.

Examples:
  msgfy auth login              # Interactive login with email/password
  msgfy auth login --api-key    # Login with API key
  msgfy auth logout             # Clear stored credentials
  msgfy auth whoami             # Show current user information`,
}

var (
	apiKey string
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Linktor",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiKey != "" {
			// API key authentication
			return loginWithAPIKey(apiKey)
		}

		// Interactive login
		return loginInteractive()
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		credPath := filepath.Join(home, ".msgfy", "credentials")
		if err := os.Remove(credPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		// Also clear from viper
		viper.Set("api_key", "")
		viper.Set("access_token", "")

		if err := saveConfig(); err != nil {
			warn("Could not update config file: %v", err)
		}

		success("Logged out successfully")
		return nil
	},
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user information",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		user, err := c.GetCurrentUser()
		if err != nil {
			return fmt.Errorf("not authenticated. Run: msgfy auth login")
		}

		fmt.Printf("Email:  %s\n", user.Email)
		fmt.Printf("Name:   %s\n", user.Name)
		fmt.Printf("Tenant: %s\n", user.TenantName)
		fmt.Printf("Role:   %s\n", user.Role)

		return nil
	},
}

var authTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "List API tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		tokens, err := c.ListAPIKeys()
		if err != nil {
			return err
		}

		if len(tokens) == 0 {
			info("No API tokens found")
			return nil
		}

		// TODO: Render tokens table
		for _, t := range tokens {
			fmt.Printf("  %s  %s  %s\n", t.ID, t.Name, t.CreatedAt)
		}

		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authWhoamiCmd)
	authCmd.AddCommand(authTokensCmd)

	authLoginCmd.Flags().StringVar(&apiKey, "api-key", "", "Login with API key instead of email/password")
}

func loginWithAPIKey(key string) error {
	// Validate the API key by making a test request
	c, err := client.NewWithAPIKey(key)
	if err != nil {
		return err
	}

	user, err := c.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("invalid API key")
	}

	// Save credentials
	if err := saveCredentials("api_key", key); err != nil {
		return err
	}

	// Also save to config
	viper.Set("api_key", key)
	if err := saveConfig(); err != nil {
		warn("Could not update config file: %v", err)
	}

	success("Logged in as %s (%s)", user.Email, user.TenantName)
	return nil
}

func loginInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	// Get email
	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	email = strings.TrimSpace(email)

	// Get password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	fmt.Println()
	password := string(passwordBytes)

	// Authenticate
	c, err := client.NewAnonymous()
	if err != nil {
		return err
	}

	loginResp, err := c.Login(email, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Save credentials
	creds := map[string]string{
		"access_token":  loginResp.AccessToken,
		"refresh_token": loginResp.RefreshToken,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	credPath := filepath.Join(home, ".msgfy", "credentials")
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return err
	}

	success("Logged in successfully")
	info("Token saved to ~/.msgfy/credentials")

	return nil
}

func saveCredentials(key, value string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	credPath := filepath.Join(home, ".msgfy", "credentials")

	creds := make(map[string]string)

	// Read existing credentials
	if data, err := os.ReadFile(credPath); err == nil {
		json.Unmarshal(data, &creds)
	}

	creds[key] = value

	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	return os.WriteFile(credPath, data, 0600)
}
