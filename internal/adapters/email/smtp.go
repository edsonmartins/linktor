package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SMTPProvider implements the EmailProvider interface using SMTP
type SMTPProvider struct {
	config *Config

	// Sem prazo, um servidor que aceita a conexão e nunca responde prendia o
	// envio além dos 60 s de AckWait da fila, e a mensagem era reentregue e
	// enviada duas vezes. Campos (e não constantes) para o teste encurtar.
	timeoutConexao time.Duration
	timeoutIO      time.Duration
}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider(config *Config) (*SMTPProvider, error) {
	return &SMTPProvider{
		config:         config,
		timeoutConexao: 10 * time.Second,
		timeoutIO:      30 * time.Second,
	}, nil
}

// ValidateConfig validates the SMTP configuration
func (p *SMTPProvider) ValidateConfig() error {
	if p.config.SMTPHost == "" {
		return fmt.Errorf("smtp_host is required")
	}
	if p.config.SMTPPort == 0 {
		return fmt.Errorf("smtp_port is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("from_email is required")
	}
	return nil
}

// GetProviderName returns the provider name
func (p *SMTPProvider) GetProviderName() string {
	return string(ProviderSMTP)
}

// conectar abre a sessão SMTP pronta para enviar: conexão com prazo, TLS
// conforme a configuração e autenticação quando há credencial.
//
// É o único caminho de conexão. Antes havia três (TLS, STARTTLS e o SendMail
// da biblioteca) e o Testar Conexão usava um quarto, que não fazia STARTTLS
// oportunista: com criptografia "none" o teste falhava enquanto o envio real
// passava. Com "none" o comportamento é o do net/smtp.SendMail: sobe para TLS
// quando o servidor oferece.
func (p *SMTPProvider) conectar(ctx context.Context) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)
	discador := net.Dialer{Timeout: p.timeoutConexao}
	conn, err := discador.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(p.timeoutIO))

	tlsCfg := &tls.Config{ServerName: p.config.SMTPHost}
	if p.config.SMTPEncryption == "tls" {
		conn = tls.Client(conn, tlsCfg)
	}

	client, err := smtp.NewClient(conn, p.config.SMTPHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	switch p.config.SMTPEncryption {
	case "tls":
	case "starttls":
		if err := client.StartTLS(tlsCfg); err != nil {
			client.Close()
			return nil, fmt.Errorf("STARTTLS failed: %w", err)
		}
	default:
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				client.Close()
				return nil, fmt.Errorf("STARTTLS failed: %w", err)
			}
		}
	}

	if p.config.SMTPUsername != "" && p.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", p.config.SMTPUsername, p.config.SMTPPassword, p.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			client.Close()
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}
	return client, nil
}

// TestConnection tests the SMTP connection
func (p *SMTPProvider) TestConnection(ctx context.Context) error {
	client, err := p.conectar(ctx)
	if err != nil {
		return err
	}
	return client.Quit()
}

// Send sends an email via SMTP. Erro de transporte ou do servidor volta como
// erro (com o código SMTP dentro), para quem chama decidir se repete.
func (p *SMTPProvider) Send(ctx context.Context, email *OutboundEmail) (*SendResult, error) {
	if len(email.To) == 0 {
		return &SendResult{
			Success:   false,
			Error:     "no recipients specified",
			Permanent: true,
			Timestamp: time.Now(),
		}, nil
	}

	// Generate message ID
	messageID := fmt.Sprintf("<%s@%s>", uuid.New().String(), extractDomain(p.config.FromEmail))
	msg := p.buildMessage(email, messageID)

	recipients := append([]string{}, email.To...)
	recipients = append(recipients, email.CC...)
	recipients = append(recipients, email.BCC...)

	client, err := p.conectar(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Mail(p.config.FromEmail); err != nil {
		return nil, fmt.Errorf("MAIL command failed: %w", err)
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return nil, fmt.Errorf("RCPT command failed for %s: %w", recipient, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return nil, fmt.Errorf("DATA command failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close message: %w", err)
	}
	// O servidor já aceitou a mensagem; um QUIT que falhe não a desfaz.
	_ = client.Quit()

	return &SendResult{
		Success:   true,
		MessageID: messageID,
		Timestamp: time.Now(),
	}, nil
}

// buildMessage builds the MIME message
func (p *SMTPProvider) buildMessage(email *OutboundEmail, messageID string) []byte {
	var sb strings.Builder

	// Headers. Assunto e nome saem codificados (RFC 2047): antes iam em UTF-8
	// cru, o que alguns servidores recusam e alguns clientes mostram errado.
	if fromName := p.config.FromName; fromName != "" {
		sb.WriteString(fmt.Sprintf("From: %s <%s>\r\n", codificarCabecalho(fromName), p.config.FromEmail))
	} else {
		sb.WriteString(fmt.Sprintf("From: %s\r\n", p.config.FromEmail))
	}

	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))

	if len(email.CC) > 0 {
		sb.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.CC, ", ")))
	}

	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", codificarCabecalho(email.Subject)))
	sb.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString("MIME-Version: 1.0\r\n")

	if email.ReplyTo != "" {
		sb.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
	}

	if email.InReplyTo != "" {
		sb.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", email.InReplyTo))
	}

	if email.References != "" {
		sb.WriteString(fmt.Sprintf("References: %s\r\n", email.References))
	}

	// Custom headers
	for k, v := range email.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	if len(email.Attachments) == 0 {
		escreverCorpo(&sb, email)
		return []byte(sb.String())
	}

	boundary := fmt.Sprintf("----=_Part_%s", uuid.New().String())
	sb.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	sb.WriteString("\r\n")

	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	escreverCorpo(&sb, email)

	for _, att := range email.Attachments {
		nome := codificarCabecalho(att.Filename)
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.ContentType, nome))
		sb.WriteString("Content-Transfer-Encoding: base64\r\n")
		sb.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", nome))
		if att.ContentID != "" {
			sb.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", att.ContentID))
		}
		sb.WriteString("\r\n")
		sb.WriteString(encodeBase64(att.Content))
		sb.WriteString("\r\n")
	}

	sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return []byte(sb.String())
}

// escreverCorpo escreve a parte de texto (HTML quando há) em quoted-printable.
// Antes o cabeçalho declarava quoted-printable e o texto ia cru, e a versão
// sem anexo nem declarava codificação; os dois ramos eram cópias.
func escreverCorpo(sb *strings.Builder, email *OutboundEmail) {
	tipo, corpo := "text/plain", email.TextBody
	if email.HTMLBody != "" {
		tipo, corpo = "text/html", email.HTMLBody
	}
	sb.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n", tipo))
	sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	w := quotedprintable.NewWriter(sb)
	_, _ = w.Write([]byte(corpo))
	_ = w.Close()
	sb.WriteString("\r\n")
}

// codificarCabecalho aplica RFC 2047 só quando há caractere fora do ASCII.
func codificarCabecalho(s string) string {
	return mime.QEncoding.Encode("utf-8", s)
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return "localhost"
}

// encodeBase64 encodes bytes to base64 with line wrapping
func encodeBase64(data []byte) string {
	encoded := make([]byte, base64EncodedLen(len(data)))
	base64Encode(encoded, data)

	// Wrap lines at 76 characters
	var sb strings.Builder
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		sb.Write(encoded[i:end])
		sb.WriteString("\r\n")
	}
	return sb.String()
}

// base64EncodedLen returns the length of the base64 encoding of an input of length n
func base64EncodedLen(n int) int {
	return (n + 2) / 3 * 4
}

// base64Encode encodes src using standard base64 encoding
func base64Encode(dst, src []byte) {
	const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	di, si := 0, 0
	n := (len(src) / 3) * 3
	for si < n {
		val := uint(src[si+0])<<16 | uint(src[si+1])<<8 | uint(src[si+2])
		dst[di+0] = encodeStd[val>>18&0x3F]
		dst[di+1] = encodeStd[val>>12&0x3F]
		dst[di+2] = encodeStd[val>>6&0x3F]
		dst[di+3] = encodeStd[val&0x3F]
		si += 3
		di += 4
	}

	remain := len(src) - si
	if remain == 0 {
		return
	}

	val := uint(src[si+0]) << 16
	if remain == 2 {
		val |= uint(src[si+1]) << 8
	}

	dst[di+0] = encodeStd[val>>18&0x3F]
	dst[di+1] = encodeStd[val>>12&0x3F]

	switch remain {
	case 2:
		dst[di+2] = encodeStd[val>>6&0x3F]
		dst[di+3] = '='
	case 1:
		dst[di+2] = '='
		dst[di+3] = '='
	}
}
