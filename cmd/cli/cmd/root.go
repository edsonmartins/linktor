package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	profile       string
	outputFormat  string
	noColor       bool

	// Version info (set at build time)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "msgfy",
	Short: "Linktor CLI - Manage your multichannel messaging platform",
	Long: `msgfy is the official command-line interface for the Linktor platform.

It allows you to manage channels, send messages, handle conversations,
and administer your self-hosted Linktor server.

Get started:
  msgfy auth login              # Authenticate with your account
  msgfy channel list            # List your channels
  msgfy send --help             # Learn how to send messages

Documentation: https://docs.linktor.io/cli`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.msgfy/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "use a specific profile")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "output format (table, json, yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	// Bind to viper
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("output_format", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(channelCmd)
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(convCmd)
	rootCmd.AddCommand(contactCmd)
	rootCmd.AddCommand(botCmd)
	rootCmd.AddCommand(flowCmd)
	rootCmd.AddCommand(kbCmd)
	rootCmd.AddCommand(webhookCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(serverCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".msgfy")

		// Create config directory if it doesn't exist
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "Error creating config directory:", err)
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("MSGFY")
	viper.AutomaticEnv()

	// Default values
	viper.SetDefault("base_url", "https://api.linktor.io")
	viper.SetDefault("output_format", "table")
	viper.SetDefault("color", true)
	viper.SetDefault("page_size", 20)
	viper.SetDefault("timeout", "30s")

	// Read config file
	if err := viper.ReadInConfig(); err == nil {
		// If profile specified, merge profile settings
		if p := viper.GetString("profile"); p != "" {
			loadProfile(p)
		} else if dp := viper.GetString("default_profile"); dp != "" {
			loadProfile(dp)
		}
	}

	// Handle color setting
	if noColor || !viper.GetBool("color") {
		color.NoColor = true
	}
}

func loadProfile(name string) {
	profiles := viper.GetStringMap("profiles")
	if profileData, ok := profiles[name]; ok {
		if pd, ok := profileData.(map[string]interface{}); ok {
			for key, value := range pd {
				viper.Set(key, value)
			}
		}
	}
}

// Helper functions for colored output
func success(format string, a ...interface{}) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s %s\n", green("✓"), fmt.Sprintf(format, a...))
}

func errorMsg(format string, a ...interface{}) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s %s\n", red("✗"), fmt.Sprintf(format, a...))
}

func info(format string, a ...interface{}) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s %s\n", cyan("ℹ"), fmt.Sprintf(format, a...))
}

func warn(format string, a ...interface{}) {
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s %s\n", yellow("⚠"), fmt.Sprintf(format, a...))
}
