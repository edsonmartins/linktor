package service

import (
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// emailReplyMetadata preenche, sem sobrescrever o que o caller mandou, os
// metadados que fazem a resposta entrar na thread do cliente: o assunto com
// "Re:" e In-Reply-To/References apontando para a última mensagem recebida.
// Sem isto a resposta saía como um email solto chamado "Message".
//
// O id é guardado sem colchetes (IMAP e webhooks tiram), e o SMTP escreve os
// cabeçalhos como estão, então os colchetes entram aqui. Um caller da API ainda
// pode mandar subject/in_reply_to/references e eles prevalecem.
func emailReplyMetadata(last *entity.Message, meta map[string]string) {
	if meta["subject"] == "" {
		if assunto := strings.TrimSpace(last.Metadata["subject"]); assunto != "" {
			if !strings.HasPrefix(strings.ToLower(assunto), "re:") {
				assunto = "Re: " + assunto
			}
			meta["subject"] = assunto
		}
	}

	id := strings.Trim(last.ExternalID, "<>")
	if id == "" {
		return
	}
	id = "<" + id + ">"
	if meta["in_reply_to"] == "" {
		meta["in_reply_to"] = id
	}
	if meta["references"] == "" {
		meta["references"] = strings.TrimSpace(last.Metadata["references"] + " " + id)
	}
}
