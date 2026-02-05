package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Manage bots",
	Long: `Create, view, and manage bots.

Examples:
  msgfy bot list
  msgfy bot show bt_abc123
  msgfy bot create --name "Support Bot" --agent ag_xyz
  msgfy bot start bt_abc123
  msgfy bot stop bt_abc123
  msgfy bot logs bt_abc123 --follow`,
}

var (
	botName     string
	botAgentID  string
	botChannels string
	botFollow   bool
	botLimit    int
)

var botListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bots",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if botLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", botLimit)
		}

		resp, err := c.ListBots(params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No bots found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Name", "Agent", "Status", "Conversations"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, bot := range resp.Data {
			agentID := ""
			if bot.AgentID != "" {
				if len(bot.AgentID) > 12 {
					agentID = bot.AgentID[:12] + "..."
				} else {
					agentID = bot.AgentID
				}
			}
			table.Append([]string{
				bot.ID,
				bot.Name,
				agentID,
				bot.Status,
				fmt.Sprintf("%d active", bot.ActiveConversations),
			})
		}

		table.Render()
		return nil
	},
}

var botShowCmd = &cobra.Command{
	Use:   "show <bot-id>",
	Short: "Show bot details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		bot, err := c.GetBot(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(bot, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:           %s\n", bot.ID)
		fmt.Printf("Name:         %s\n", bot.Name)
		fmt.Printf("Agent:        %s\n", bot.AgentID)
		fmt.Printf("Status:       %s\n", bot.Status)
		fmt.Printf("Enabled:      %v\n", bot.Enabled)
		fmt.Printf("Channels:     %s\n", strings.Join(bot.ChannelIDs, ", "))
		fmt.Printf("Active Conv:  %d\n", bot.ActiveConversations)
		fmt.Printf("Created:      %s\n", bot.CreatedAt.Format("2006-01-02 15:04"))

		return nil
	},
}

var botCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new bot",
	RunE: func(cmd *cobra.Command, args []string) error {
		if botName == "" {
			return fmt.Errorf("--name is required")
		}
		if botAgentID == "" {
			return fmt.Errorf("--agent is required")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		input := map[string]interface{}{
			"name":    botName,
			"agentId": botAgentID,
			"enabled": true,
		}

		if botChannels != "" {
			input["channelIds"] = strings.Split(botChannels, ",")
		}

		bot, err := c.CreateBot(input)
		if err != nil {
			return err
		}

		success("Bot created: %s", bot.ID)
		info("Start with: msgfy bot start %s", bot.ID)

		return nil
	},
}

var botUpdateCmd = &cobra.Command{
	Use:   "update <bot-id>",
	Short: "Update a bot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		input := make(map[string]interface{})

		if botName != "" {
			input["name"] = botName
		}
		if botAgentID != "" {
			input["agentId"] = botAgentID
		}
		if botChannels != "" {
			input["channelIds"] = strings.Split(botChannels, ",")
		}

		if len(input) == 0 {
			return fmt.Errorf("no updates specified")
		}

		_, err = c.UpdateBot(args[0], input)
		if err != nil {
			return err
		}

		success("Bot updated")
		return nil
	},
}

var botStartCmd = &cobra.Command{
	Use:   "start <bot-id>",
	Short: "Start a bot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		bot, err := c.StartBot(args[0])
		if err != nil {
			return err
		}

		success("Bot started: %s", bot.Status)
		return nil
	},
}

var botStopCmd = &cobra.Command{
	Use:   "stop <bot-id>",
	Short: "Stop a bot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		bot, err := c.StopBot(args[0])
		if err != nil {
			return err
		}

		success("Bot stopped: %s", bot.Status)
		return nil
	},
}

var botStatusCmd = &cobra.Command{
	Use:   "status <bot-id>",
	Short: "Show bot status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		bot, err := c.GetBot(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			status := map[string]interface{}{
				"id":                  bot.ID,
				"name":                bot.Name,
				"status":              bot.Status,
				"activeConversations": bot.ActiveConversations,
				"enabled":             bot.Enabled,
			}
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Bot: %s (%s)\n", bot.Name, bot.ID)
		fmt.Printf("Status: %s\n", bot.Status)
		fmt.Printf("Agent: %s\n", bot.AgentID)
		fmt.Printf("Channels: %s\n", strings.Join(bot.ChannelIDs, ", "))
		fmt.Printf("Active conversations: %d\n", bot.ActiveConversations)

		return nil
	},
}

var botLogsCmd = &cobra.Command{
	Use:   "logs <bot-id>",
	Short: "Show bot logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		botID := args[0]

		// Get initial logs
		logs, err := c.GetBotLogs(botID, map[string]string{
			"limit": fmt.Sprintf("%d", botLimit),
		})
		if err != nil {
			return err
		}

		// Print logs
		for _, log := range logs {
			printBotLog(log)
		}

		if !botFollow {
			return nil
		}

		// Follow mode - poll for new logs
		info("Following logs... (Ctrl+C to stop)")

		lastTimestamp := time.Now()
		if len(logs) > 0 {
			lastTimestamp = logs[len(logs)-1].Timestamp
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sigChan:
				fmt.Println()
				return nil
			case <-ticker.C:
				newLogs, err := c.GetBotLogs(botID, map[string]string{
					"since": lastTimestamp.Format(time.RFC3339),
				})
				if err != nil {
					continue
				}

				for _, log := range newLogs {
					printBotLog(log)
					lastTimestamp = log.Timestamp
				}
			}
		}
	},
}

var botDeleteCmd = &cobra.Command{
	Use:   "delete <bot-id>",
	Short: "Delete a bot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmDelete {
			return fmt.Errorf("use --confirm to delete the bot")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		if err := c.DeleteBot(args[0]); err != nil {
			return err
		}

		success("Bot deleted")
		return nil
	},
}

func init() {
	botCmd.AddCommand(botListCmd)
	botCmd.AddCommand(botShowCmd)
	botCmd.AddCommand(botCreateCmd)
	botCmd.AddCommand(botUpdateCmd)
	botCmd.AddCommand(botStartCmd)
	botCmd.AddCommand(botStopCmd)
	botCmd.AddCommand(botStatusCmd)
	botCmd.AddCommand(botLogsCmd)
	botCmd.AddCommand(botDeleteCmd)

	// List flags
	botListCmd.Flags().IntVar(&botLimit, "limit", 20, "Limit results")

	// Create flags
	botCreateCmd.Flags().StringVar(&botName, "name", "", "Bot name (required)")
	botCreateCmd.Flags().StringVar(&botAgentID, "agent", "", "Agent ID (required)")
	botCreateCmd.Flags().StringVar(&botChannels, "channels", "", "Channel IDs (comma-separated)")

	// Update flags
	botUpdateCmd.Flags().StringVar(&botName, "name", "", "Bot name")
	botUpdateCmd.Flags().StringVar(&botAgentID, "agent", "", "Agent ID")
	botUpdateCmd.Flags().StringVar(&botChannels, "channels", "", "Channel IDs (comma-separated)")

	// Logs flags
	botLogsCmd.Flags().BoolVarP(&botFollow, "follow", "f", false, "Follow logs")
	botLogsCmd.Flags().IntVar(&botLimit, "limit", 50, "Number of log entries")

	// Delete flags
	botDeleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion")
}

func printBotLog(log client.BotLog) {
	timestamp := log.Timestamp.Format("15:04:05")
	level := log.Level

	// Color based on level
	switch level {
	case "error":
		fmt.Printf("[%s] \033[31m%s\033[0m %s\n", timestamp, level, log.Message)
	case "warn":
		fmt.Printf("[%s] \033[33m%s\033[0m %s\n", timestamp, level, log.Message)
	case "info":
		fmt.Printf("[%s] \033[34m%s\033[0m %s\n", timestamp, level, log.Message)
	default:
		fmt.Printf("[%s] %s %s\n", timestamp, level, log.Message)
	}
}
