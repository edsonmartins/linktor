package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var convCmd = &cobra.Command{
	Use:     "conv",
	Aliases: []string{"conversation", "conversations"},
	Short:   "Manage conversations",
	Long: `View and manage conversations.

Examples:
  msgfy conv list --status open
  msgfy conv show cv_abc123
  msgfy conv messages cv_abc123
  msgfy conv close cv_abc123`,
}

var (
	convStatus    string
	convChannelID string
	convLimit     int
	convSince     string
)

var convListCmd = &cobra.Command{
	Use:   "list",
	Short: "List conversations",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if convStatus != "" {
			params["status"] = convStatus
		}
		if convChannelID != "" {
			params["channelId"] = convChannelID
		}
		if convLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", convLimit)
		}

		resp, err := c.ListConversations(params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No conversations found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Contact", "Channel", "Status", "Last Message"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, conv := range resp.Data {
			lastMsg := ""
			if conv.LastMessage != nil {
				lastMsg = timeAgo(conv.LastMessage.CreatedAt)
			}
			table.Append([]string{
				conv.ID,
				conv.ContactName,
				conv.ChannelID[:8] + "...",
				conv.Status,
				lastMsg,
			})
		}

		table.Render()
		return nil
	},
}

var convShowCmd = &cobra.Command{
	Use:   "show <conversation-id>",
	Short: "Show conversation details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		conv, err := c.GetConversation(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(conv, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Conversation: %s\n", conv.ID)
		fmt.Printf("Contact:      %s\n", conv.ContactName)
		fmt.Printf("Channel:      %s\n", conv.ChannelID)
		fmt.Printf("Status:       %s\n", conv.Status)
		fmt.Printf("Created:      %s\n", conv.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Messages:     %d\n", conv.MessageCount)

		return nil
	},
}

var convMessagesCmd = &cobra.Command{
	Use:   "messages <conversation-id>",
	Short: "Show conversation messages",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if convLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", convLimit)
		}

		resp, err := c.GetConversationMessages(args[0], params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No messages found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Print messages in reverse order (oldest first)
		for i := len(resp.Data) - 1; i >= 0; i-- {
			msg := resp.Data[i]
			direction := "←"
			if msg.Direction == "outbound" {
				direction = "→"
			}
			fmt.Printf("[%s] %s %s\n",
				msg.CreatedAt.Format("15:04"),
				direction,
				msg.Text,
			)
		}

		return nil
	},
}

var convCloseCmd = &cobra.Command{
	Use:     "close <conversation-id>",
	Aliases: []string{"resolve"},
	Short:   "Close/resolve a conversation",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		_, err = c.CloseConversation(args[0])
		if err != nil {
			return err
		}

		success("Conversation closed")
		return nil
	},
}

var convReopenCmd = &cobra.Command{
	Use:   "reopen <conversation-id>",
	Short: "Reopen a closed conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		_, err = c.ReopenConversation(args[0])
		if err != nil {
			return err
		}

		success("Conversation reopened")
		return nil
	},
}

var convExportCmd = &cobra.Command{
	Use:   "export <conversation-id>",
	Short: "Export conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		conv, err := c.GetConversation(args[0])
		if err != nil {
			return err
		}

		msgs, err := c.GetConversationMessages(args[0], map[string]string{"limit": "1000"})
		if err != nil {
			return err
		}

		export := map[string]interface{}{
			"conversation": conv,
			"messages":     msgs.Data,
		}

		data, _ := json.MarshalIndent(export, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

func init() {
	convCmd.AddCommand(convListCmd)
	convCmd.AddCommand(convShowCmd)
	convCmd.AddCommand(convMessagesCmd)
	convCmd.AddCommand(convCloseCmd)
	convCmd.AddCommand(convReopenCmd)
	convCmd.AddCommand(convExportCmd)

	// List flags
	convListCmd.Flags().StringVar(&convStatus, "status", "", "Filter by status (open, closed)")
	convListCmd.Flags().StringVar(&convChannelID, "channel", "", "Filter by channel ID")
	convListCmd.Flags().IntVar(&convLimit, "limit", 20, "Limit results")

	// Messages flags
	convMessagesCmd.Flags().IntVar(&convLimit, "limit", 50, "Limit messages")
	convMessagesCmd.Flags().StringVar(&convSince, "since", "", "Show messages since (e.g., '1h', '24h')")
}

func timeAgo(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(d.Hours()/24))
}
