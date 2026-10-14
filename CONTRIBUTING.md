# Como contribuir com o Linktor

Contribuições são bem-vindas: correções, novos canais, documentação, testes. Este guia explica como preparar o ambiente, o que um PR precisa ter e como ele é revisado.

## Antes de começar

- Para correções pequenas, abra o PR direto.
- Para mudanças grandes (um canal novo, mudança de API ou de banco), abra antes uma [issue](https://github.com/edsonmartins/linktor/issues) descrevendo a proposta. Assim ninguém perde trabalho com algo que não vai entrar.
- **Falhas de segurança não vão em issue pública.** Use [Report a vulnerability](https://github.com/edsonmartins/linktor/security/advisories/new) na aba Security do repositório.

## Preparando o ambiente

1. Faça um fork do repositório e clone o seu fork.
2. Siga a seção [Instalação](README.md#instalação) do README: ela explica como subir tudo em Docker ou rodar backend e admin na máquina, e traz o `.env` completo.
3. Crie uma branch a partir da `main`:

```bash
git switch -c fix/descricao-curta
```

## Antes de abrir o PR

Rode o que o CI vai rodar:

```bash
# Backend
go vet ./...
go build ./...
make test

# Admin (se mexeu em web/admin)
cd web/admin
npm ci
npx tsc --noEmit
```

Os testes de migração e os e2e do admin rodam no CI; rodá-los localmente é opcional.

Inclua testes para o que mudou. Correção de bug vem com um teste que falhava antes e passa depois.

## Commits

Seguimos [Conventional Commits](https://www.conventionalcommits.org/), com a descrição em português:

```
fix(telegram): webhook deixa de ser descartado como duplicado
feat(slack): respostas passam a sair na thread de origem
docs(readme): corrige o passo a passo de instalação
```

Tipos usados: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`. O escopo entre parênteses é a área do código (um canal, `db`, `admin`, `conversas`...).

No corpo do commit, explique o porquê: o que estava errado e o que a mudança resolve. O diff já mostra o como.

### Assinatura (DCO)

Todo commit de contribuição precisa da linha `Signed-off-by`, que declara que você tem o direito de enviar aquele código sob a licença do projeto, nos termos do [Developer Certificate of Origin](https://developercertificate.org/). Basta commitar com `-s`:

```bash
git commit -s -m "fix(rcs): contato passa a ser identificado pelo telefone"
```

Esqueceu em commits já feitos? Assine todos de uma vez e atualize o PR:

```bash
git rebase --signoff main
git push --force-with-lease
```

Um check chamado **DCO** confere a assinatura em cada PR.

## O PR

- Um assunto por PR. Correções independentes vão em PRs separados.
- Descreva o problema, a solução e como você testou. O modelo de PR já traz essas seções.
- Mudança visual no admin: inclua print ou gif.
- Mudança que exige variável de ambiente nova: atualize o `.env.example` e a seção de instalação do README.

No seu primeiro PR, o CI só roda depois que um mantenedor aprovar a execução. É uma proteção padrão do GitHub para contas novas no repositório; a partir do segundo PR ele roda sozinho.

O merge acontece depois do CI verde e da revisão de um mantenedor.

## Adicionando um canal

Cada canal vive em `internal/adapters/<canal>/`. O Direto (`internal/adapters/direto`) é o menor e serve de ponto de partida.

1. **Envio:** implemente `outbound.Sender` e um `outbound.Factory` (`internal/outbound/sender.go`) e registre a factory em `cmd/server/main.go`, junto das outras (`outboundResolver.Register`).
2. **Recebimento:** crie o handler do webhook em `internal/api/handlers/` e a rota em `cmd/server/main.go`, no grupo `/api/v1/webhooks`. Verifique a assinatura do provedor antes de processar.
3. **Tipo de canal:** acrescente o tipo em `internal/domain/entity/channel.go`.
4. **Admin:** crie o formulário em `web/admin/src/app/(dashboard)/channels/`, inclua o tipo na lista de `channels/page.tsx` e em `web/admin/src/types`, e escreva os textos nos três idiomas de `web/admin/src/i18n/locales/`.
5. **Documentação:** acrescente o canal ao [roteiro de testes](docs/roteiro-testes-canais.md).

## Licença

O Linktor é distribuído sob a [Apache License 2.0](LICENSE). Ao contribuir, você concorda que sua contribuição seja licenciada nos mesmos termos.
