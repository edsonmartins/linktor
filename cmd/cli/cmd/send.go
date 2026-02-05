package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message",
	Long: `Send a message through a channel.

Examples:
  msgfy send --channel ch_abc123 --to "+5544999999999" --text "Hello!"
  msgfy send --channel ch_abc123 --to "+5544999999999" --image "https://example.com/img.jpg"
  msgfy send --channel ch_abc123 --to-file contacts.txt --text "Broadcast message"
  msgfy send -i                    # Interactive mode`,
	RunE: runSend,
}

var (
	sendChannel  string
	sendTo       string
	sendToFile   string
	sendText     string
	sendImage    string
	sendDocument string
	sendCaption  string
	sendFilename string
	sendDelay    time.Duration
	interactive  bool
)

func init() {
	sendCmd.Flags().StringVar(&sendChannel, "channel", "", "Channel ID to send from (required)")
	sendCmd.Flags().StringVar(&sendTo, "to", "", "Recipient (phone number or identifier)")
	sendCmd.Flags().StringVar(&sendToFile, "to-file", "", "File with recipient list (one per line)")
	sendCmd.Flags().StringVar(&sendText, "text", "", "Message text")
	sendCmd.Flags().StringVar(&sendImage, "image", "", "Image URL or path")
	sendCmd.Flags().StringVar(&sendDocument, "document", "", "Document path")
	sendCmd.Flags().StringVar(&sendCaption, "caption", "", "Caption for media")
	sendCmd.Flags().StringVar(&sendFilename, "filename", "", "Filename for document")
	sendCmd.Flags().DurationVar(&sendDelay, "delay", 0, "Delay between messages (for broadcast)")
	sendCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")
}

func runSend(cmd *cobra.Command, args []string) error {
	if interactive {
		return runInteractiveSend()
	}

	if sendChannel == "" {
		return fmt.Errorf("--channel is required")
	}

	if sendTo == "" && sendToFile == "" {
		return fmt.Errorf("--to or --to-file is required")
	}

	if sendText == "" && sendImage == "" && sendDocument == "" {
		return fmt.Errorf("message content is required (--text, --image, or --document)")
	}

	c, err := client.New()
	if err != nil {
		return err
	}

	// Get recipients
	var recipients []string
	if sendToFile != "" {
		recipients, err = readRecipientsFile(sendToFile)
		if err != nil {
			return fmt.Errorf("failed to read recipients file: %w", err)
		}
	} else {
		recipients = []string{sendTo}
	}

	// Build message input
	input := buildMessageInput()

	// Send to each recipient
	sent := 0
	failed := 0

	for i, recipient := range recipients {
		// For now, we need a conversation - this is simplified
		// In a real implementation, you'd either find or create a conversation
		msgInput := map[string]interface{}{
			"channelId": sendChannel,
			"to":        recipient,
		}
		for k, v := range input {
			msgInput[k] = v
		}

		// TODO: Implement direct send API or find/create conversation
		fmt.Printf("Sending to %s...\n", recipient)

		// This is a placeholder - actual implementation would use the API
		msg, err := sendDirectMessage(c, sendChannel, recipient, input)
		if err != nil {
			errorMsg("Failed to send to %s: %v", recipient, err)
			failed++
			continue
		}

		success("Message sent: %s", msg.ID)
		sent++

		// Delay between messages for broadcast
		if sendDelay > 0 && i < len(recipients)-1 {
			time.Sleep(sendDelay)
		}
	}

	if len(recipients) > 1 {
		fmt.Printf("\nSummary: %d sent, %d failed\n", sent, failed)
	}

	return nil
}

func runInteractiveSend() error {
	reader := bufio.NewReader(os.Stdin)
	c, err := client.New()
	if err != nil {
		return err
	}

	// Get channel if not provided
	if sendChannel == "" {
		fmt.Print("Channel ID: ")
		channel, _ := reader.ReadString('\n')
		sendChannel = strings.TrimSpace(channel)
	}

	// Get recipient
	fmt.Print("To: ")
	to, _ := reader.ReadString('\n')
	to = strings.TrimSpace(to)

	fmt.Println("Type your message (press Enter to send, 'quit' to exit):")

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if text == "quit" || text == "exit" {
			break
		}

		if text == "" {
			continue
		}

		input := map[string]interface{}{
			"text": text,
		}

		msg, err := sendDirectMessage(c, sendChannel, to, input)
		if err != nil {
			errorMsg("Failed to send: %v", err)
			continue
		}

		success("Sent: %s", msg.ID)
	}

	return nil
}

func buildMessageInput() map[string]interface{} {
	input := make(map[string]interface{})

	if sendText != "" {
		input["text"] = sendText
	}

	if sendImage != "" {
		input["contentType"] = "image"
		input["media"] = map[string]interface{}{
			"type": "image",
			"url":  sendImage,
		}
		if sendCaption != "" {
			input["text"] = sendCaption
		}
	}

	if sendDocument != "" {
		input["contentType"] = "document"
		input["media"] = map[string]interface{}{
			"type":     "document",
			"url":      sendDocument,
			"filename": sendFilename,
		}
		if sendCaption != "" {
			input["text"] = sendCaption
		}
	}

	return input
}

func readRecipientsFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var recipients []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			recipients = append(recipients, line)
		}
	}

	return recipients, scanner.Err()
}

// sendDirectMessage sends a message directly (simplified implementation)
// In practice, this would either use a direct send API or find/create conversation
func sendDirectMessage(c *client.Client, channelID, to string, input map[string]interface{}) (*client.Message, error) {
	// This is a placeholder implementation
	// The actual implementation would:
	// 1. Find or create a conversation for this channel/recipient
	// 2. Send the message to that conversation

	// For now, we'll return a mock response
	// In real implementation, you'd call the appropriate API endpoint
	return &client.Message{
		ID:        "msg_placeholder",
		Direction: "outbound",
		Status:    "sent",
	}, nil
}
