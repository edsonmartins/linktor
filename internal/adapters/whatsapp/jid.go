package whatsapp

import (
	"strings"

	"go.mau.fi/whatsmeow/types"
)

// resolveRecipientJID turns a recipient string into a WhatsApp JID.
//
// It exists because whatsmeow's types.ParseJID does the WRONG thing for a bare
// phone number: given "554488122990" (no "@"), ParseJID returns
// JID{User: "", Server: "554488122990"} with a nil error — the number lands in
// the Server field. Sending to that JID fails with "can't send message to
// unknown server 554488122990". The common `jid, err := types.ParseJID(to); if
// err != nil { jid = types.NewJID(...) }` guard never triggers because ParseJID
// does not return an error in this case.
//
// So: a value without "@" is treated as a user and given the default user
// server (s.whatsapp.net); anything already containing "@" (a full JID, possibly
// with device/agent suffixes) is parsed by whatsmeow as before.
func resolveRecipientJID(to string) (types.JID, error) {
	if !strings.ContainsRune(to, '@') {
		return types.NewJID(to, types.DefaultUserServer), nil
	}
	return types.ParseJID(to)
}
