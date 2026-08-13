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
	// sendMetadata are repeatable key=value pairs preserved end to end on the
	// message; sendIdempotencyKey is the shorthand for the one key that changes
	// the API's behaviour rather than just riding along.
	sendMetadata       []string
	sendIdempotencyKey string
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
	sendCmd.Flags().StringArrayVar(&sendMetadata, "metadata", nil, "Metadata carried with the message, key=value (repeatable)")
	sendCmd.Flags().StringVar(&sendIdempotencyKey, "idempotency-key", "", "Logical send key: repeating it returns the original message instead of sending twice")
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

	// An idempotency key names ONE logical message. Reused across a broadcast it
	// would send to the first recipient and hand back that same message for all
	// the others, so refuse the combination instead of silently dropping sends.
	if len(recipients) > 1 && buildSendMetadata()["idempotency_key"] != "" {
		return fmt.Errorf("an idempotency key identifies a single message; do not combine it with a recipient list")
	}

	// Build message input
	input := buildMessageInput()

	// Send to each recipient
	sent := 0
	failed := 0

	for i, recipient := range recipients {
		fmt.Printf("Sending to %s...\n", recipient)

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

// sendDirectMessage posts to the channel+recipient send route
// (POST /api/v1/messages/send), which resolves the contact and conversation
// server-side. The route currently carries text only; media still has to go
// through a conversation, so we say so instead of silently dropping the
// attachment.
func sendDirectMessage(c *client.Client, channelID, to string, input map[string]interface{}) (*client.DirectSendResult, error) {
	if input["media"] != nil {
		return nil, fmt.Errorf("direct send carries text only; send media from an existing conversation (POST /api/v1/conversations/{id}/messages)")
	}
	if metadata := buildSendMetadata(); len(metadata) > 0 {
		input["metadata"] = metadata
	}
	return c.SendDirectMessage(channelID, to, input)
}

// buildSendMetadata assembles the metadata carried end to end with the message:
// the repeatable --metadata k=v pairs plus --idempotency-key, which makes a
// repeated send return the original message instead of delivering twice.
func buildSendMetadata() map[string]string {
	metadata := make(map[string]string, len(sendMetadata)+1)
	for _, pair := range sendMetadata {
		key, value, found := strings.Cut(pair, "=")
		if !found {
			continue
		}
		if key = strings.TrimSpace(key); key != "" {
			metadata[key] = value
		}
	}
	if sendIdempotencyKey != "" {
		metadata["idempotency_key"] = sendIdempotencyKey
	}
	return metadata
}
