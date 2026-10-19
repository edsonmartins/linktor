package email

import (
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// Texto devolve o corpo legível: a parte text/plain quando existe, senão o
// HTML convertido. Um email só-HTML chegava ao chat com as tags cruas.
func (e *IncomingEmail) Texto() string {
	if strings.TrimSpace(e.TextBody) != "" {
		return e.TextBody
	}
	return htmlParaTexto(e.HTMLBody)
}

// Tags cujo conteúdo não é texto para ler.
var tagsIgnoradas = map[string]bool{"script": true, "style": true, "head": true, "noscript": true}

// Tags que separam linhas.
var tagsDeBloco = map[string]bool{
	"p": true, "div": true, "br": true, "li": true, "tr": true, "blockquote": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "table": true, "pre": true, "hr": true,
	"section": true, "article": true, "header": true, "footer": true,
}

// htmlParaTexto extrai o texto visível de um HTML: descarta script, style e
// head, troca blocos por quebra de linha, células por espaço, e descarta linhas
// em branco. As entidades (&amp;, &nbsp;…) já vêm decodificadas do tokenizador.
func htmlParaTexto(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}

	var b strings.Builder
	t := html.NewTokenizer(strings.NewReader(s))
	ignorando := 0
	for {
		switch t.Next() {
		case html.ErrorToken:
			return limparLinhas(b.String())
		case html.StartTagToken, html.SelfClosingTagToken:
			nome, _ := t.TagName()
			tag := string(nome)
			if tagsIgnoradas[tag] {
				ignorando++
				continue
			}
			switch {
			case tagsDeBloco[tag]:
				b.WriteByte('\n')
			case tag == "td" || tag == "th":
				b.WriteByte(' ')
			}
		case html.EndTagToken:
			nome, _ := t.TagName()
			tag := string(nome)
			if tagsIgnoradas[tag] {
				if ignorando > 0 {
					ignorando--
				}
				continue
			}
			if tagsDeBloco[tag] {
				b.WriteByte('\n')
			}
		case html.TextToken:
			if ignorando == 0 {
				b.WriteString(colapsarEspacos(string(t.Text())))
			}
		}
	}
}

// colapsarEspacos reduz qualquer sequência de espaço (inclusive quebras de
// linha do código-fonte HTML, que não são quebras visíveis) a um espaço.
func colapsarEspacos(s string) string {
	var b strings.Builder
	emEspaco := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !emEspaco {
				b.WriteByte(' ')
				emEspaco = true
			}
			continue
		}
		b.WriteRune(r)
		emEspaco = false
	}
	return b.String()
}

// limparLinhas apara cada linha e descarta as vazias: cada bloco do HTML vira
// uma linha, como o texto alternativo que os clientes de email geram.
func limparLinhas(s string) string {
	var linhas []string
	for _, linha := range strings.Split(s, "\n") {
		if linha = strings.TrimSpace(linha); linha != "" {
			linhas = append(linhas, linha)
		}
	}
	return strings.Join(linhas, "\n")
}
