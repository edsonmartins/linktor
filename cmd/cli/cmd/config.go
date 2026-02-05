package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long: `Manage msgfy CLI configuration including API keys, profiles, and settings.

Configuration is stored in ~/.msgfy/config.yaml`,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := viper.AllSettings()

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Key", "Value"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for key, value := range settings {
			// Mask sensitive values
			displayValue := fmt.Sprintf("%v", value)
			if strings.Contains(strings.ToLower(key), "key") ||
				strings.Contains(strings.ToLower(key), "token") ||
				strings.Contains(strings.ToLower(key), "secret") {
				if len(displayValue) > 8 {
					displayValue = displayValue[:4] + "***" + displayValue[len(displayValue)-4:]
				}
			}
			table.Append([]string{key, displayValue})
		}

		table.Render()
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := viper.Get(key)
		if value == nil {
			return fmt.Errorf("configuration key '%s' not found", key)
		}

		fmt.Println(value)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		viper.Set(key, value)

		if err := saveConfig(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		success("Configuration '%s' updated", key)
		return nil
	},
}

var configUseCmd = &cobra.Command{
	Use:   "use <profile>",
	Short: "Switch to a different profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		profiles := viper.GetStringMap("profiles")
		if _, ok := profiles[profileName]; !ok {
			return fmt.Errorf("profile '%s' not found", profileName)
		}

		viper.Set("default_profile", profileName)

		if err := saveConfig(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		success("Switched to profile '%s'", profileName)
		return nil
	},
}

var configProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage configuration profiles",
}

var configProfileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles := viper.GetStringMap("profiles")
		defaultProfile := viper.GetString("default_profile")

		if len(profiles) == 0 {
			info("No profiles configured. Create one with: msgfy config profile create <name>")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Base URL", "Default"})
		table.SetBorder(false)

		for name, data := range profiles {
			baseURL := ""
			if pd, ok := data.(map[string]interface{}); ok {
				if bu, ok := pd["base_url"].(string); ok {
					baseURL = bu
				}
			}

			isDefault := ""
			if name == defaultProfile {
				isDefault = "✓"
			}

			table.Append([]string{name, baseURL, isDefault})
		}

		table.Render()
		return nil
	},
}

var (
	profileAPIKey  string
	profileBaseURL string
)

var configProfileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		profiles := viper.GetStringMap("profiles")
		if profiles == nil {
			profiles = make(map[string]interface{})
		}

		profileData := map[string]interface{}{}

		if profileAPIKey != "" {
			profileData["api_key"] = profileAPIKey
		}
		if profileBaseURL != "" {
			profileData["base_url"] = profileBaseURL
		} else {
			profileData["base_url"] = "https://api.linktor.io"
		}

		profiles[profileName] = profileData
		viper.Set("profiles", profiles)

		if err := saveConfig(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		success("Profile '%s' created", profileName)
		info("Switch to it with: msgfy config use %s", profileName)
		return nil
	},
}

var configProfileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		profiles := viper.GetStringMap("profiles")
		if _, ok := profiles[profileName]; !ok {
			return fmt.Errorf("profile '%s' not found", profileName)
		}

		delete(profiles, profileName)
		viper.Set("profiles", profiles)

		// Clear default if it was this profile
		if viper.GetString("default_profile") == profileName {
			viper.Set("default_profile", "")
		}

		if err := saveConfig(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		success("Profile '%s' deleted", profileName)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		if cfgFile := viper.ConfigFileUsed(); cfgFile != "" {
			fmt.Println(cfgFile)
		} else {
			home, _ := os.UserHomeDir()
			fmt.Println(filepath.Join(home, ".msgfy", "config.yaml"))
		}
	},
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configProfileCmd)
	configCmd.AddCommand(configPathCmd)

	configProfileCmd.AddCommand(configProfileListCmd)
	configProfileCmd.AddCommand(configProfileCreateCmd)
	configProfileCmd.AddCommand(configProfileDeleteCmd)

	// Profile create flags
	configProfileCreateCmd.Flags().StringVar(&profileAPIKey, "api-key", "", "API key for the profile")
	configProfileCreateCmd.Flags().StringVar(&profileBaseURL, "base-url", "", "Base URL for the profile")
}

func saveConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".msgfy", "config.yaml")

	// Get all settings
	settings := viper.AllSettings()

	// Marshal to YAML
	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0600)
}
