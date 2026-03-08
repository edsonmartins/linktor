package cmd

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// timeAgo tests (conversation.go)
// =============================================================================

func TestTimeAgo_JustNow(t *testing.T) {
	result := timeAgo(time.Now())
	assert.Equal(t, "just now", result)
}

func TestTimeAgo_MinutesAgo(t *testing.T) {
	result := timeAgo(time.Now().Add(-5 * time.Minute))
	assert.Equal(t, "5 min ago", result)
}

func TestTimeAgo_OneMinuteAgo(t *testing.T) {
	result := timeAgo(time.Now().Add(-1 * time.Minute))
	assert.Equal(t, "1 min ago", result)
}

func TestTimeAgo_HoursAgo(t *testing.T) {
	result := timeAgo(time.Now().Add(-3 * time.Hour))
	assert.Equal(t, "3 hours ago", result)
}

func TestTimeAgo_DaysAgo(t *testing.T) {
	result := timeAgo(time.Now().Add(-2 * 24 * time.Hour))
	assert.Equal(t, "2 days ago", result)
}

func TestTimeAgo_30SecondsAgo(t *testing.T) {
	result := timeAgo(time.Now().Add(-30 * time.Second))
	assert.Equal(t, "just now", result)
}

// =============================================================================
// buildMessageInput tests (send.go)
// =============================================================================

func resetSendVars() {
	sendText = ""
	sendImage = ""
	sendDocument = ""
	sendCaption = ""
	sendFilename = ""
}

func TestBuildMessageInput_TextOnly(t *testing.T) {
	resetSendVars()
	defer resetSendVars()

	sendText = "hello"
	input := buildMessageInput()

	assert.Equal(t, "hello", input["text"])
	assert.Nil(t, input["contentType"])
	assert.Nil(t, input["media"])
}

func TestBuildMessageInput_ImageWithCaption(t *testing.T) {
	resetSendVars()
	defer resetSendVars()

	sendImage = "https://example.com/img.jpg"
	sendCaption = "Nice photo"

	input := buildMessageInput()

	assert.Equal(t, "image", input["contentType"])
	assert.Equal(t, "Nice photo", input["text"])

	media, ok := input["media"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "image", media["type"])
	assert.Equal(t, "https://example.com/img.jpg", media["url"])
}

func TestBuildMessageInput_DocumentWithFilename(t *testing.T) {
	resetSendVars()
	defer resetSendVars()

	sendDocument = "/path/to/doc.pdf"
	sendFilename = "report.pdf"
	sendCaption = "Monthly report"

	input := buildMessageInput()

	assert.Equal(t, "document", input["contentType"])
	assert.Equal(t, "Monthly report", input["text"])

	media, ok := input["media"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "document", media["type"])
	assert.Equal(t, "/path/to/doc.pdf", media["url"])
	assert.Equal(t, "report.pdf", media["filename"])
}

func TestBuildMessageInput_Empty(t *testing.T) {
	resetSendVars()
	defer resetSendVars()

	input := buildMessageInput()
	assert.Empty(t, input)
}

func TestBuildMessageInput_ImageWithoutCaption(t *testing.T) {
	resetSendVars()
	defer resetSendVars()

	sendImage = "https://example.com/img.jpg"

	input := buildMessageInput()

	assert.Equal(t, "image", input["contentType"])
	assert.Nil(t, input["text"])
}

// =============================================================================
// readRecipientsFile tests (send.go)
// =============================================================================

func TestReadRecipientsFile_ValidFile(t *testing.T) {
	f, err := os.CreateTemp("", "recipients-*.txt")
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	_, _ = f.WriteString("+5544999999999\n+5544888888888\n+5544777777777\n")
	f.Close()

	recipients, err := readRecipientsFile(f.Name())
	assert.NoError(t, err)
	assert.Equal(t, []string{"+5544999999999", "+5544888888888", "+5544777777777"}, recipients)
}

func TestReadRecipientsFile_CommentsAndBlanks(t *testing.T) {
	f, err := os.CreateTemp("", "recipients-*.txt")
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	_, _ = f.WriteString("# This is a comment\n+5544999999999\n\n# Another comment\n+5544888888888\n  \n")
	f.Close()

	recipients, err := readRecipientsFile(f.Name())
	assert.NoError(t, err)
	assert.Equal(t, []string{"+5544999999999", "+5544888888888"}, recipients)
}

func TestReadRecipientsFile_NonexistentFile(t *testing.T) {
	_, err := readRecipientsFile("/nonexistent/path/file.txt")
	assert.Error(t, err)
}

func TestReadRecipientsFile_EmptyFile(t *testing.T) {
	f, err := os.CreateTemp("", "recipients-*.txt")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	f.Close()

	recipients, err := readRecipientsFile(f.Name())
	assert.NoError(t, err)
	assert.Empty(t, recipients)
}

// =============================================================================
// parseMapping tests (contact.go)
// =============================================================================

func TestParseMapping_EmptyString(t *testing.T) {
	result := parseMapping("", []string{"Name", "Phone"})
	assert.Empty(t, result)
}

func TestParseMapping_ValidMapping(t *testing.T) {
	result := parseMapping("name:Nome,phone:Telefone", []string{"Nome", "Telefone"})
	assert.Equal(t, "name", result["Nome"])
	assert.Equal(t, "phone", result["Telefone"])
}

func TestParseMapping_SingleMapping(t *testing.T) {
	result := parseMapping("name:FullName", []string{"FullName"})
	assert.Equal(t, "name", result["FullName"])
	assert.Len(t, result, 1)
}

func TestParseMapping_InvalidEntry(t *testing.T) {
	// An entry without ":" is simply skipped (SplitN returns 1 element)
	result := parseMapping("invalidentry,name:Nome", []string{"Nome"})
	assert.Equal(t, "name", result["Nome"])
	assert.Len(t, result, 1)
}

// =============================================================================
// truncateURL tests (webhook.go)
// =============================================================================

func TestTruncateURL_ShortURL(t *testing.T) {
	url := "https://example.com"
	result := truncateURL(url, 40)
	assert.Equal(t, url, result)
}

func TestTruncateURL_ExactLength(t *testing.T) {
	url := "https://example.com/webhook" // 28 chars
	result := truncateURL(url, 28)
	assert.Equal(t, url, result)
}

func TestTruncateURL_LongURL(t *testing.T) {
	url := "https://very-long-domain.example.com/api/webhooks/v1/receiver/endpoint"
	result := truncateURL(url, 30)
	assert.Equal(t, 30, len(result))
	assert.True(t, result[len(result)-3:] == "...")
	assert.Equal(t, url[:27]+"...", result)
}

// =============================================================================
// generateSamplePayload tests (webhook.go)
// =============================================================================

func TestGenerateSamplePayload_MessageReceived(t *testing.T) {
	payload := generateSamplePayload("message.received")

	data, ok := payload["data"].(map[string]interface{})
	assert.True(t, ok, "payload should have data field")
	assert.NotEmpty(t, data["id"])
	assert.NotEmpty(t, data["text"])
	assert.Equal(t, "inbound", data["direction"])
}

func TestGenerateSamplePayload_ConversationCreated(t *testing.T) {
	payload := generateSamplePayload("conversation.created")

	data, ok := payload["data"].(map[string]interface{})
	assert.True(t, ok, "payload should have data field")
	assert.NotEmpty(t, data["contactName"])
	assert.Equal(t, "open", data["status"])
}

func TestGenerateSamplePayload_MessageSent(t *testing.T) {
	payload := generateSamplePayload("message.sent")

	data, ok := payload["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "outbound", data["direction"])
	assert.Equal(t, "sent", data["status"])
}

func TestGenerateSamplePayload_ConversationClosed(t *testing.T) {
	payload := generateSamplePayload("conversation.closed")

	data, ok := payload["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "closed", data["status"])
}

func TestGenerateSamplePayload_UnknownType(t *testing.T) {
	payload := generateSamplePayload("some.unknown.event")

	data, ok := payload["data"].(map[string]interface{})
	assert.True(t, ok, "payload should have data field")
	assert.Contains(t, data["message"], "some.unknown.event")
}

// =============================================================================
// jsonReader tests (webhook.go)
// =============================================================================

func TestJsonReader_ReadFull(t *testing.T) {
	data := []byte(`{"test": "hello"}`)
	reader := &jsonReader{data: data}

	buf := make([]byte, 100)
	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buf[:n])
}

func TestJsonReader_EOF(t *testing.T) {
	data := []byte(`{"test": "hello"}`)
	reader := &jsonReader{data: data}

	// First read consumes all data
	buf := make([]byte, 100)
	_, _ = reader.Read(buf)

	// Second read should return EOF
	n, err := reader.Read(buf)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

func TestJsonReader_SmallBuffer(t *testing.T) {
	data := []byte("abcdefghij")
	reader := &jsonReader{data: data}

	buf := make([]byte, 4)

	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, []byte("abcd"), buf[:n])

	n, err = reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, []byte("efgh"), buf[:n])

	n, err = reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte("ij"), buf[:n])

	n, err = reader.Read(buf)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

func TestJsonReader_EmptyData(t *testing.T) {
	reader := &jsonReader{data: []byte{}}

	buf := make([]byte, 10)
	n, err := reader.Read(buf)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

// =============================================================================
// colorStatus tests (server.go)
// =============================================================================

func TestColorStatus_Connected(t *testing.T) {
	result := colorStatus("connected")
	assert.Contains(t, result, "\033[32m") // green escape code
	assert.Contains(t, result, "connected")
	assert.Contains(t, result, "\033[0m") // reset
}

func TestColorStatus_Healthy(t *testing.T) {
	result := colorStatus("healthy")
	assert.Contains(t, result, "\033[32m")
	assert.Contains(t, result, "healthy")
}

func TestColorStatus_Ready(t *testing.T) {
	result := colorStatus("ready")
	assert.Contains(t, result, "\033[32m")
}

func TestColorStatus_Disconnected(t *testing.T) {
	result := colorStatus("disconnected")
	assert.Contains(t, result, "\033[31m") // red escape code
	assert.Contains(t, result, "disconnected")
}

func TestColorStatus_Unhealthy(t *testing.T) {
	result := colorStatus("unhealthy")
	assert.Contains(t, result, "\033[31m")
}

func TestColorStatus_Error(t *testing.T) {
	result := colorStatus("error")
	assert.Contains(t, result, "\033[31m")
}

func TestColorStatus_Unknown(t *testing.T) {
	result := colorStatus("unknown")
	assert.Contains(t, result, "\033[33m") // yellow escape code
	assert.Contains(t, result, "unknown")
}

func TestColorStatus_Pending(t *testing.T) {
	result := colorStatus("pending")
	assert.Contains(t, result, "\033[33m") // yellow for unrecognized status
}
