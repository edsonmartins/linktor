package email

import "strings"

// Helpers de endereço compartilhados pelo webhook de provedor e pelo leitor
// IMAP. Ficavam no cliente IMAP artesanal; sobreviveram à remoção dele porque
// o caminho de webhook (Mailgun/SendGrid/SES/Postmark) continua usando.

// parseFromName extracts the name from a From header
func parseFromName(from string) string {
	// Parse "Name <email@example.com>" format
	if idx := strings.Index(from, "<"); idx > 0 {
		return strings.TrimSpace(from[:idx])
	}
	return ""
}

// parseAddressList parses a comma-separated list of email addresses
func parseAddressList(list string) []string {
	addresses := strings.Split(list, ",")
	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			// Extract email from "Name <email>" format
			if idx := strings.Index(addr, "<"); idx >= 0 {
				if endIdx := strings.Index(addr, ">"); endIdx > idx {
					addr = addr[idx+1 : endIdx]
				}
			}
			result = append(result, addr)
		}
	}
	return result
}
