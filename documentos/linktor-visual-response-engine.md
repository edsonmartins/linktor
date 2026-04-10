# Linktor Visual Response Engine (VRE)

## Arquitetura Atual

O VRE do Linktor renderiza respostas visuais a partir de **templates SVG** e retorna imagens prontas para canais que não têm bons componentes interativos nativos, como WhatsApp não oficial.

Fluxo:

1. O bot ou MCP envia `template_id` + `data`, ou um SVG customizado em `svg`.
2. O registry resolve `templates/tenants/{tenant_id}/{template_id}.svg`.
3. Se não houver override do tenant, usa `templates/default/{template_id}.svg`.
4. O template SVG é processado com Go `text/template`.
5. O renderer abre o SVG em headless Chrome via `data:image/svg+xml`.
6. O VRE captura a imagem em PNG ou JPEG.
7. A API retorna `image_base64`, `mime_type`, `caption`, dimensões e metadata de render.

O caminho legado de markup de página foi removido. Uploads e renderizações customizadas devem enviar SVG.

## Endpoints

### Renderizar

```http
POST /api/v1/vre/render
Content-Type: application/json
Authorization: Bearer <token>
```

Com template:

```json
{
  "tenant_id": "demo-tenant",
  "template_id": "menu_opcoes",
  "data": {
    "titulo": "Como posso ajudar?",
    "opcoes": [
      { "label": "Fazer pedido", "descricao": "Monte seu pedido" },
      { "label": "Consultar entrega", "descricao": "Acompanhe o status" }
    ]
  },
  "channel": "whatsapp",
  "format": "png"
}
```

Com SVG customizado:

```json
{
  "tenant_id": "demo-tenant",
  "svg": "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"800\" height=\"600\"><rect width=\"800\" height=\"600\" fill=\"#0B1220\"/><text x=\"40\" y=\"80\" fill=\"#fff\" font-size=\"32\">Olá</text></svg>",
  "format": "png"
}
```

Resposta:

```json
{
  "success": true,
  "data": {
    "image_base64": "iVBORw0KGgo...",
    "mime_type": "image/png",
    "format": "png",
    "width": 800,
    "height": 600,
    "caption": "Como posso ajudar?\n1. Fazer pedido\n2. Consultar entrega",
    "rendered_at": "2026-04-08T12:00:00Z"
  }
}
```

### Renderizar e enviar

```http
POST /api/v1/vre/render-and-send
Content-Type: application/json
Authorization: Bearer <token>
```

```json
{
  "conversation_id": "conv_123",
  "template_id": "status_pedido",
  "data": {
    "numero_pedido": "4587",
    "status_atual": "transporte",
    "valor_total": 189.9
  },
  "caption": "Seu pedido saiu para entrega."
}
```

### Listar templates

```http
GET /api/v1/vre/templates?tenant_id=demo-tenant
Authorization: Bearer <token>
```

### Preview

```http
GET /api/v1/vre/templates/{template_id}/preview
Authorization: Bearer <token>
```

### Upload de template SVG

```http
POST /api/v1/vre/templates/{template_id}
Content-Type: text/plain
Authorization: Bearer <token>

<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600">...</svg>
```

O backend valida que o conteúdo enviado é SVG. O arquivo é persistido como `templates/tenants/{tenant_id}/{template_id}.svg`.

## Templates

Templates padrão atuais:

- `menu_opcoes.svg`
- `status_pedido.svg`
- `card_produto.svg`
- `lista_produtos.svg`
- `confirmacao.svg`
- `cobranca_pix.svg`

Estrutura:

```text
templates/
├── default/
│   ├── menu_opcoes.svg
│   ├── status_pedido.svg
│   ├── card_produto.svg
│   ├── lista_produtos.svg
│   ├── confirmacao.svg
│   └── cobranca_pix.svg
└── tenants/
    └── {tenant_id}/
        ├── config.json
        └── {template_id}.svg
```

Exemplo mínimo:

```svg
<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600" viewBox="0 0 800 600">
  <rect width="800" height="600" rx="24" fill="{{ .Brand.Background }}"/>
  <text x="48" y="84" fill="{{ .Brand.PrimaryColor }}" font-size="32" font-weight="700">
    {{ .Data.titulo }}
  </text>
  {{ range $i, $opcao := .Data.opcoes }}
  <text x="64" y="{{ add 150 (mul $i 64) }}" fill="{{ $.Brand.TextColor }}" font-size="24">
    {{ add $i 1 }}. {{ $opcao.label }}
  </text>
  {{ end }}
</svg>
```

## Branding

O registry injeta configuração de tenant quando houver `templates/tenants/{tenant_id}/config.json`.

Campos principais de `.Brand`:

- `Name`
- `LogoURL`
- `PrimaryColor`
- `SecondaryColor`
- `AccentColor`
- `Background`
- `TextColor`
- `FontFamily`
- `BorderRadius`

## Funções De Template

Funções disponíveis nos SVGs:

- `add`, `sub`, `mul`, `div`, `mod`
- `formatCurrency`, `formatDate`, `formatUnit`
- `truncate`, `default`
- `icon`, `statusColor`, `stockStatus`
- `percentage`, `dict`, `list`, `json`

## Formatos

Formato padrão: `png`.

Formatos suportados no fluxo atual:

- `png`
- `jpeg`

`webp` não deve ser anunciado por clientes enquanto não houver encoder real para esse formato.

## Contrato Com MCP

As tools MCP de VRE chamam os mesmos endpoints e devem enviar apenas templates SVG resolvidos pelo backend ou dados estruturados para os templates padrão. O default de `format` é `png`.
