package email

import "testing"

func TestHTMLParaTexto(t *testing.T) {
	casos := []struct{ nome, html, quer string }{
		{"parágrafos", `<p>Olá, <b>Alexandre</b>!</p><p>Linha 2</p>`, "Olá, Alexandre!\nLinha 2"},
		{"entidades", `<p>Tom &amp; Jerry &lt;3&nbsp;!</p>`, "Tom & Jerry <3 !"},
		{"style e script", `<style>p{}</style><script>x()</script><p>Texto</p>`, "Texto"},
		{"lista", `<ul><li>Um</li><li>Dois</li></ul>`, "Um\nDois"},
		{"br", `Linha<br>Quebra<br/>Fim`, "Linha\nQuebra\nFim"},
		{"tabela", `<table><tr><td>Produto</td><td>Preço</td></tr><tr><td>Fone</td><td>R$ 10</td></tr></table>`, "Produto Preço\nFone R$ 10"},
		{"citação do Gmail", `<div>Resposta</div><div class="gmail_quote">Em seg., escreveu:<blockquote>Original</blockquote></div>`, "Resposta\nEm seg., escreveu:\nOriginal"},
		{"quebras do código-fonte não são quebras", "<div>\n  Texto em\n  duas linhas do fonte\n</div>", "Texto em duas linhas do fonte"},
		{"vazio", "  ", ""},
		{"Outlook sem texto", `<html><head><meta charset="utf-8"><style>P{margin:0}</style></head><body><div><br></div></body></html>`, ""},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := htmlParaTexto(c.html); got != c.quer {
				t.Errorf("htmlParaTexto(%q)\n got: %q\nquer: %q", c.html, got, c.quer)
			}
		})
	}
}

func TestIncomingEmailTextoPrefereAParteDeTexto(t *testing.T) {
	e := &IncomingEmail{TextBody: "texto puro", HTMLBody: "<p>html</p>"}
	if e.Texto() != "texto puro" {
		t.Errorf("Texto() = %q", e.Texto())
	}
	e = &IncomingEmail{TextBody: "  \n", HTMLBody: "<p>só html</p>"}
	if e.Texto() != "só html" {
		t.Errorf("Texto() = %q", e.Texto())
	}
}
