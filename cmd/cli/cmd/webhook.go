package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage and test webhooks",
	Long: `Test, simulate, and debug webhooks.

Examples:
  msgfy webhook list
  msgfy webhook test https://example.com/webhook
  msgfy webhook simulate message.received --url https://example.com/webhook
  msgfy webhook events --limit 20
  msgfy webhook listen --port 3000`,
}

var (
	webhookURL       string
	webhookPort      int
	webhookEvent     string
	webhookData      string
	webhookLimit     int
)

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured webhooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		resp, err := c.ListWebhooks(map[string]string{})
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No webhooks configured")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "URL", "Events", "Status"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, wh := range resp.Data {
			events := fmt.Sprintf("%d events", len(wh.Events))
			if len(wh.Events) <= 3 {
				events = ""
				for i, e := range wh.Events {
					if i > 0 {
						events += ", "
					}
					events += e
				}
			}

			status := "active"
			if !wh.Enabled {
				status = "disabled"
			}

			table.Append([]string{
				wh.ID,
				truncateURL(wh.URL, 40),
				events,
				status,
			})
		}

		table.Render()
		return nil
	},
}

var webhookTestCmd = &cobra.Command{
	Use:   "test <url>",
	Short: "Test webhook endpoint connectivity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		fmt.Printf("Testing webhook %s...\n", url)

		// Create test payload
		testPayload := map[string]interface{}{
			"type":      "test",
			"timestamp": time.Now().Format(time.RFC3339),
			"data": map[string]interface{}{
				"message": "Webhook test from msgfy CLI",
			},
		}

		payloadBytes, _ := json.Marshal(testPayload)

		// Send request
		start := time.Now()
		resp, err := http.Post(url, "application/json",
			io.NopCloser(
				&jsonReader{data: payloadBytes},
			))
		duration := time.Since(start)

		if err != nil {
			errorMsg("Connection failed: %v", err)
			return nil
		}
		defer resp.Body.Close()

		// Read response
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			success("Endpoint reachable (%d %s)", resp.StatusCode, http.StatusText(resp.StatusCode))
		} else {
			errorMsg("Endpoint returned %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		}

		fmt.Printf("Response time: %dms\n", duration.Milliseconds())

		if len(body) > 0 && outputFormat == "json" {
			fmt.Printf("Response body:\n%s\n", string(body))
		}

		return nil
	},
}

var webhookSimulateCmd = &cobra.Command{
	Use:   "simulate <event-type>",
	Short: "Simulate a webhook event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if webhookURL == "" {
			return fmt.Errorf("--url is required")
		}

		eventType := args[0]

		// Build event payload
		var payload map[string]interface{}

		if webhookData != "" {
			if err := json.Unmarshal([]byte(webhookData), &payload); err != nil {
				return fmt.Errorf("invalid JSON data: %w", err)
			}
		} else {
			// Generate sample payload based on event type
			payload = generateSamplePayload(eventType)
		}

		payload["type"] = eventType
		payload["timestamp"] = time.Now().Format(time.RFC3339)

		payloadBytes, _ := json.MarshalIndent(payload, "", "  ")

		fmt.Printf("Simulating event: %s\n", eventType)
		fmt.Printf("Payload:\n%s\n\n", string(payloadBytes))

		// Send request
		start := time.Now()
		resp, err := http.Post(webhookURL, "application/json",
			io.NopCloser(&jsonReader{data: payloadBytes}))
		duration := time.Since(start)

		if err != nil {
			errorMsg("Failed to send: %v", err)
			return nil
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			success("Event sent (%d %s)", resp.StatusCode, http.StatusText(resp.StatusCode))
		} else {
			errorMsg("Endpoint returned %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		}

		fmt.Printf("Response time: %dms\n", duration.Milliseconds())

		if len(body) > 0 {
			fmt.Printf("Response: %s\n", string(body))
		}

		return nil
	},
}

var webhookEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show recent webhook events",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if webhookLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", webhookLimit)
		}

		events, err := c.ListWebhookEvents(params)
		if err != nil {
			return err
		}

		if len(events) == 0 {
			info("No recent webhook events")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(events, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Timestamp", "Event", "Status", "Duration"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, e := range events {
			status := fmt.Sprintf("%d", e.StatusCode)
			if e.StatusCode >= 200 && e.StatusCode < 300 {
				status = fmt.Sprintf("\033[32m%d\033[0m", e.StatusCode)
			} else {
				status = fmt.Sprintf("\033[31m%d\033[0m", e.StatusCode)
			}

			table.Append([]string{
				e.Timestamp.Format("15:04:05"),
				e.EventType,
				status,
				fmt.Sprintf("%dms", e.Duration),
			})
		}

		table.Render()
		return nil
	},
}

var webhookListenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Start a local webhook listener for debugging",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf(":%d", webhookPort)

		fmt.Printf("Starting webhook listener on http://localhost%s\n", addr)
		info("Press Ctrl+C to stop")
		fmt.Println()

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			timestamp := time.Now().Format("15:04:05")

			// Read body
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()

			// Parse JSON if possible
			var payload interface{}
			if err := json.Unmarshal(body, &payload); err == nil {
				prettyBody, _ := json.MarshalIndent(payload, "  ", "  ")
				body = prettyBody
			}

			// Log the request
			fmt.Printf("[%s] %s %s\n", timestamp, r.Method, r.URL.Path)

			// Log headers
			if r.Header.Get("X-Linktor-Event") != "" {
				fmt.Printf("  Event: %s\n", r.Header.Get("X-Linktor-Event"))
			}
			if r.Header.Get("X-Linktor-Signature") != "" {
				fmt.Printf("  Signature: %s\n", r.Header.Get("X-Linktor-Signature")[:20]+"...")
			}

			// Log body
			if len(body) > 0 {
				fmt.Printf("  Body:\n  %s\n", string(body))
			}

			fmt.Println()

			// Respond with OK
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		return http.ListenAndServe(addr, nil)
	},
}

func init() {
	webhookCmd.AddCommand(webhookListCmd)
	webhookCmd.AddCommand(webhookTestCmd)
	webhookCmd.AddCommand(webhookSimulateCmd)
	webhookCmd.AddCommand(webhookEventsCmd)
	webhookCmd.AddCommand(webhookListenCmd)

	// Simulate flags
	webhookSimulateCmd.Flags().StringVar(&webhookURL, "url", "", "Webhook URL (required)")
	webhookSimulateCmd.Flags().StringVar(&webhookData, "data", "", "Custom JSON payload")

	// Events flags
	webhookEventsCmd.Flags().IntVar(&webhookLimit, "limit", 20, "Number of events to show")

	// Listen flags
	webhookListenCmd.Flags().IntVar(&webhookPort, "port", 3000, "Port to listen on")
}

// jsonReader implements io.Reader for a byte slice
type jsonReader struct {
	data []byte
	pos  int
}

func (r *jsonReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

func generateSamplePayload(eventType string) map[string]interface{} {
	switch eventType {
	case "message.received":
		return map[string]interface{}{
			"data": map[string]interface{}{
				"id":             "msg_sample123",
				"conversationId": "cv_sample456",
				"direction":      "inbound",
				"text":           "Hello, this is a test message",
				"contentType":    "text",
				"createdAt":      time.Now().Format(time.RFC3339),
			},
		}
	case "message.sent":
		return map[string]interface{}{
			"data": map[string]interface{}{
				"id":             "msg_sample789",
				"conversationId": "cv_sample456",
				"direction":      "outbound",
				"text":           "This is a response message",
				"contentType":    "text",
				"status":         "sent",
				"createdAt":      time.Now().Format(time.RFC3339),
			},
		}
	case "conversation.created":
		return map[string]interface{}{
			"data": map[string]interface{}{
				"id":          "cv_sample456",
				"channelId":   "ch_sample123",
				"contactId":   "ct_sample789",
				"contactName": "John Doe",
				"status":      "open",
				"createdAt":   time.Now().Format(time.RFC3339),
			},
		}
	case "conversation.closed":
		return map[string]interface{}{
			"data": map[string]interface{}{
				"id":        "cv_sample456",
				"status":    "closed",
				"closedAt":  time.Now().Format(time.RFC3339),
				"closedBy":  "agent",
			},
		}
	default:
		return map[string]interface{}{
			"data": map[string]interface{}{
				"message": fmt.Sprintf("Sample payload for %s", eventType),
			},
		}
	}
}
