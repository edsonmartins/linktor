package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage messaging channels",
	Long: `Manage your messaging channels (WhatsApp, Telegram, SMS, etc.)

Examples:
  msgfy channel list                          # List all channels
  msgfy channel create --type telegram --name "My Bot"
  msgfy channel config ch_abc123 --set bot_token=xxx
  msgfy channel test ch_abc123`,
}

var (
	channelType   string
	channelName   string
	channelStatus string
	channelLimit  int
	confirmDelete bool
)

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if channelType != "" {
			params["type"] = channelType
		}
		if channelStatus != "" {
			params["status"] = channelStatus
		}
		if channelLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", channelLimit)
		}

		resp, err := c.ListChannels(params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No channels found")
			return nil
		}

		// Check output format
		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Type", "Name", "Status"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, ch := range resp.Data {
			status := ch.Status
			if !ch.Enabled {
				status = "disabled"
			}
			table.Append([]string{ch.ID, ch.Type, ch.Name, status})
		}

		table.Render()
		return nil
	},
}

var channelShowCmd = &cobra.Command{
	Use:   "show <channel-id>",
	Short: "Show channel details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		ch, err := c.GetChannel(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(ch, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:        %s\n", ch.ID)
		fmt.Printf("Name:      %s\n", ch.Name)
		fmt.Printf("Type:      %s\n", ch.Type)
		fmt.Printf("Status:    %s\n", ch.Status)
		fmt.Printf("Enabled:   %v\n", ch.Enabled)
		fmt.Printf("Created:   %s\n", ch.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated:   %s\n", ch.UpdatedAt.Format("2006-01-02 15:04"))

		return nil
	},
}

var channelCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new channel",
	RunE: func(cmd *cobra.Command, args []string) error {
		if channelType == "" || channelName == "" {
			return fmt.Errorf("--type and --name are required")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		input := map[string]interface{}{
			"name":    channelName,
			"type":    channelType,
			"enabled": true,
		}

		ch, err := c.CreateChannel(input)
		if err != nil {
			return err
		}

		success("Channel created: %s", ch.ID)
		info("Configure with: msgfy channel config %s --set <key>=<value>", ch.ID)

		return nil
	},
}

var (
	configSet []string
	configGet bool
)

var channelConfigCmd = &cobra.Command{
	Use:   "config <channel-id>",
	Short: "Get or set channel configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		channelID := args[0]

		if configGet || len(configSet) == 0 {
			// Get configuration
			ch, err := c.GetChannel(channelID)
			if err != nil {
				return err
			}

			if len(ch.Config) == 0 {
				info("No configuration set")
				return nil
			}

			for key, value := range ch.Config {
				// Mask sensitive values
				displayValue := fmt.Sprintf("%v", value)
				if strings.Contains(strings.ToLower(key), "token") ||
					strings.Contains(strings.ToLower(key), "secret") ||
					strings.Contains(strings.ToLower(key), "key") {
					if len(displayValue) > 8 {
						displayValue = displayValue[:4] + "***"
					}
				}
				fmt.Printf("  %s: %s\n", key, displayValue)
			}
			return nil
		}

		// Set configuration
		ch, err := c.GetChannel(channelID)
		if err != nil {
			return err
		}

		config := ch.Config
		if config == nil {
			config = make(map[string]interface{})
		}

		for _, s := range configSet {
			parts := strings.SplitN(s, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid config format: %s (use key=value)", s)
			}
			config[parts[0]] = parts[1]
		}

		_, err = c.UpdateChannel(channelID, map[string]interface{}{
			"config": config,
		})
		if err != nil {
			return err
		}

		success("Channel configuration updated")
		return nil
	},
}

var channelTestCmd = &cobra.Command{
	Use:   "test <channel-id>",
	Short: "Test channel connectivity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		fmt.Printf("Testing channel %s...\n", args[0])

		ch, err := c.GetChannel(args[0])
		if err != nil {
			return err
		}

		if ch.Status == "connected" {
			success("Channel is connected and operational")
		} else if ch.Status == "error" {
			errorMsg("Channel has errors. Check configuration.")
		} else {
			warn("Channel status: %s", ch.Status)
		}

		return nil
	},
}

var channelConnectCmd = &cobra.Command{
	Use:   "connect <channel-id>",
	Short: "Connect a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		ch, err := c.ConnectChannel(args[0])
		if err != nil {
			return err
		}

		success("Channel connected: %s", ch.Status)
		return nil
	},
}

var channelDisconnectCmd = &cobra.Command{
	Use:   "disconnect <channel-id>",
	Short: "Disconnect a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		ch, err := c.DisconnectChannel(args[0])
		if err != nil {
			return err
		}

		success("Channel disconnected: %s", ch.Status)
		return nil
	},
}

var channelDeleteCmd = &cobra.Command{
	Use:   "delete <channel-id>",
	Short: "Delete a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmDelete {
			return fmt.Errorf("use --confirm to delete the channel")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		if err := c.DeleteChannel(args[0]); err != nil {
			return err
		}

		success("Channel deleted")
		return nil
	},
}

func init() {
	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelShowCmd)
	channelCmd.AddCommand(channelCreateCmd)
	channelCmd.AddCommand(channelConfigCmd)
	channelCmd.AddCommand(channelTestCmd)
	channelCmd.AddCommand(channelConnectCmd)
	channelCmd.AddCommand(channelDisconnectCmd)
	channelCmd.AddCommand(channelDeleteCmd)

	// List flags
	channelListCmd.Flags().StringVar(&channelType, "type", "", "Filter by channel type")
	channelListCmd.Flags().StringVar(&channelStatus, "status", "", "Filter by status")
	channelListCmd.Flags().IntVar(&channelLimit, "limit", 20, "Limit results")

	// Create flags
	channelCreateCmd.Flags().StringVar(&channelType, "type", "", "Channel type (required)")
	channelCreateCmd.Flags().StringVar(&channelName, "name", "", "Channel name (required)")

	// Config flags
	channelConfigCmd.Flags().StringArrayVar(&configSet, "set", nil, "Set config value (key=value)")
	channelConfigCmd.Flags().BoolVar(&configGet, "get", false, "Get current config")

	// Delete flags
	channelDeleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion")
}
