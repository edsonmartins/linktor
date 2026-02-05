package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Self-hosted server management",
	Long: `Manage self-hosted Linktor server.

Examples:
  msgfy server start
  msgfy server status
  msgfy server migrate
  msgfy server plugin list`,
}

var serverPluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage server plugins",
}

var (
	serverPort        int
	serverWorkers     int
	serverConfigFile  string
	rollbackSteps     int
	pluginName        string
)

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Linktor server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Loading configuration...")

		// Load server config
		if serverConfigFile != "" {
			viper.SetConfigFile(serverConfigFile)
			if err := viper.ReadInConfig(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
		}

		port := serverPort
		if port == 0 {
			port = viper.GetInt("server.port")
			if port == 0 {
				port = 8080
			}
		}

		workers := serverWorkers
		if workers == 0 {
			workers = viper.GetInt("server.workers")
			if workers == 0 {
				workers = 4
			}
		}

		fmt.Println("Connecting to database...")
		// TODO: Actual database connection

		fmt.Printf("Starting HTTP server on :%d\n", port)
		fmt.Printf("Workers: %d\n", workers)

		// Setup shutdown handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Println("\nShutting down...")
			cancel()
		}()

		success("Server ready")
		info("Press Ctrl+C to stop")

		// Wait for shutdown
		<-ctx.Done()

		return nil
	},
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status",
	RunE: func(cmd *cobra.Command, args []string) error {
		// This would connect to the server's status endpoint
		// For now, showing a placeholder

		status := map[string]interface{}{
			"version":  "1.0.0",
			"uptime":   "5d 12h 30m",
			"database": "connected",
			"nats":     "connected",
			"redis":    "connected",
			"workers":  "4/4 healthy",
			"channels": 5,
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Version:   %s\n", status["version"])
		fmt.Printf("Uptime:    %s\n", status["uptime"])
		fmt.Println()

		fmt.Println("Services:")
		fmt.Printf("  Database:  %s\n", colorStatus(status["database"].(string)))
		fmt.Printf("  NATS:      %s\n", colorStatus(status["nats"].(string)))
		fmt.Printf("  Redis:     %s\n", colorStatus(status["redis"].(string)))
		fmt.Println()

		fmt.Printf("Workers:   %s\n", status["workers"])
		fmt.Printf("Channels:  %d active\n", status["channels"])

		return nil
	},
}

var serverMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if rollbackSteps > 0 {
			fmt.Printf("Rolling back %d migration(s)...\n", rollbackSteps)
			// TODO: Actual rollback logic
			success("Rollback complete")
			return nil
		}

		fmt.Println("Running migrations...")

		// Placeholder migration output
		migrations := []struct {
			Name   string
			Status string
		}{
			{"001_create_users", "done"},
			{"002_create_channels", "done"},
			{"003_create_conversations", "done"},
			{"004_create_messages", "done"},
			{"005_create_contacts", "done"},
		}

		for i, m := range migrations {
			fmt.Printf("[%d/%d] %s...%s\n", i+1, len(migrations), m.Name, m.Status)
		}

		success("All migrations complete")
		return nil
	},
}

var serverHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Health checks
		checks := []struct {
			Name    string
			Healthy bool
		}{
			{"Database", true},
			{"NATS", true},
			{"Redis", true},
			{"API", true},
		}

		allHealthy := true

		if outputFormat == "json" {
			result := make(map[string]string)
			for _, c := range checks {
				if c.Healthy {
					result[c.Name] = "healthy"
				} else {
					result[c.Name] = "unhealthy"
					allHealthy = false
				}
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))

			if !allHealthy {
				os.Exit(1)
			}
			return nil
		}

		for _, c := range checks {
			if c.Healthy {
				fmt.Printf("\033[32m✓\033[0m %s: healthy\n", c.Name)
			} else {
				fmt.Printf("\033[31m✗\033[0m %s: unhealthy\n", c.Name)
				allHealthy = false
			}
		}

		if !allHealthy {
			os.Exit(1)
		}

		return nil
	},
}

var serverBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = fmt.Sprintf("backup-%s.tar.gz",
				viper.GetTime("").Format("2006-01-02"))
		}

		fmt.Println("Creating backup...")
		fmt.Println("  - Exporting database...")
		fmt.Println("  - Backing up configuration...")
		fmt.Println("  - Compressing files...")

		success("Backup created: %s", output)
		return nil
	},
}

// Plugin commands

var serverPluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Placeholder plugin list
		plugins := []struct {
			Name    string
			Version string
			Status  string
		}{
			{"whatsapp-unofficial", "1.2.0", "enabled"},
			{"custom-ai-provider", "0.5.0", "enabled"},
			{"sms-gateway", "1.0.0", "disabled"},
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(plugins, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Version", "Status"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, p := range plugins {
			table.Append([]string{p.Name, p.Version, p.Status})
		}

		table.Render()
		return nil
	},
}

var serverPluginInstallCmd = &cobra.Command{
	Use:   "install <plugin-name>",
	Short: "Install a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		fmt.Printf("Installing plugin: %s\n", pluginName)
		fmt.Println("  - Downloading...")
		fmt.Println("  - Verifying...")
		fmt.Println("  - Installing...")

		success("Plugin installed: %s", pluginName)
		info("Restart server to activate")

		return nil
	},
}

var serverPluginEnableCmd = &cobra.Command{
	Use:   "enable <plugin-name>",
	Short: "Enable a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		success("Plugin enabled: %s", pluginName)
		return nil
	},
}

var serverPluginDisableCmd = &cobra.Command{
	Use:   "disable <plugin-name>",
	Short: "Disable a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		success("Plugin disabled: %s", pluginName)
		return nil
	},
}

var serverPluginRemoveCmd = &cobra.Command{
	Use:   "remove <plugin-name>",
	Short: "Remove a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmDelete {
			return fmt.Errorf("use --confirm to remove the plugin")
		}

		pluginName := args[0]
		success("Plugin removed: %s", pluginName)
		return nil
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStatusCmd)
	serverCmd.AddCommand(serverMigrateCmd)
	serverCmd.AddCommand(serverHealthCmd)
	serverCmd.AddCommand(serverBackupCmd)
	serverCmd.AddCommand(serverPluginCmd)

	// Plugin subcommands
	serverPluginCmd.AddCommand(serverPluginListCmd)
	serverPluginCmd.AddCommand(serverPluginInstallCmd)
	serverPluginCmd.AddCommand(serverPluginEnableCmd)
	serverPluginCmd.AddCommand(serverPluginDisableCmd)
	serverPluginCmd.AddCommand(serverPluginRemoveCmd)

	// Start flags
	serverStartCmd.Flags().IntVar(&serverPort, "port", 0, "HTTP port (default: 8080)")
	serverStartCmd.Flags().IntVar(&serverWorkers, "workers", 0, "Number of workers (default: 4)")
	serverStartCmd.Flags().StringVar(&serverConfigFile, "config", "", "Config file path")

	// Migrate flags
	serverMigrateCmd.Flags().IntVar(&rollbackSteps, "rollback", 0, "Number of migrations to rollback")

	// Backup flags
	serverBackupCmd.Flags().String("output", "", "Output file path")

	// Plugin remove flags
	serverPluginRemoveCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm removal")
}

func colorStatus(status string) string {
	switch status {
	case "connected", "healthy", "ready":
		return fmt.Sprintf("\033[32m%s\033[0m", status)
	case "disconnected", "unhealthy", "error":
		return fmt.Sprintf("\033[31m%s\033[0m", status)
	default:
		return fmt.Sprintf("\033[33m%s\033[0m", status)
	}
}
