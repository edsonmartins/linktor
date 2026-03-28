package utils

import "strings"

// GenerateVCard generates a VCF-formatted vCard string.
func GenerateVCard(fullName, phone, email, organization string) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\n")
	b.WriteString("VERSION:3.0\n")
	b.WriteString("FN:" + fullName + "\n")
	if organization != "" {
		b.WriteString("ORG:" + organization + ";\n")
	}
	if phone != "" {
		b.WriteString("TEL;type=CELL;type=VOICE;waid=" + phone + ":" + phone + "\n")
	}
	if email != "" {
		b.WriteString("EMAIL:" + email + "\n")
	}
	b.WriteString("END:VCARD")
	return b.String()
}
