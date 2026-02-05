# msgfy - Linktor CLI

Official command-line interface for the Linktor multichannel messaging platform.

## Installation

### Go Install

```bash
go install github.com/linktor/msgfy@latest
```

### From Source

```bash
cd cmd/cli
go build -o msgfy .
```

### Homebrew (macOS)

```bash
brew install linktor/tap/msgfy
```

## Quick Start

```bash
# Authenticate
msgfy auth login

# List channels
msgfy channel list

# Send a message
msgfy send --channel ch_abc123 --to "+5544999999999" --text "Hello!"

# List conversations
msgfy conv list --status open
```

## Configuration

Configuration is stored in `~/.msgfy/config.yaml`:

```yaml
default_profile: production

profiles:
  production:
    api_key: lk_live_abc123
    base_url: https://api.linktor.io
  staging:
    api_key: lk_test_xyz789
    base_url: https://staging-api.linktor.io

settings:
  output_format: table  # table, json
  color: true
  page_size: 20
```

### Environment Variables

- `MSGFY_API_KEY` - API key for authentication
- `MSGFY_BASE_URL` - Override API base URL
- `MSGFY_PROFILE` - Default profile to use

## Commands

### Authentication

```bash
# Login with email/password
msgfy auth login

# Login with API key
msgfy auth login --api-key lk_live_abc123

# Show current user
msgfy auth whoami

# Logout
msgfy auth logout

# List API tokens
msgfy auth tokens
```

### Channels

```bash
# List channels
msgfy channel list
msgfy channel list --type whatsapp --status connected

# Show channel details
msgfy channel show ch_abc123

# Create channel
msgfy channel create --type telegram --name "Support Bot"

# Configure channel
msgfy channel config ch_abc123 --set bot_token=xxx

# Test connectivity
msgfy channel test ch_abc123

# Connect/Disconnect
msgfy channel connect ch_abc123
msgfy channel disconnect ch_abc123

# Delete channel
msgfy channel delete ch_abc123 --confirm
```

### Send Messages

```bash
# Send text message
msgfy send --channel ch_abc123 --to "+5544999999999" --text "Hello!"

# Send image
msgfy send --channel ch_abc123 --to "+5544999999999" \
  --image "https://example.com/img.jpg" \
  --caption "Check this out!"

# Send document
msgfy send --channel ch_abc123 --to "+5544999999999" \
  --document ./contract.pdf \
  --filename "Contract.pdf"

# Broadcast to multiple recipients
msgfy send --channel ch_abc123 \
  --to-file contacts.txt \
  --text "Broadcast message" \
  --delay 1s

# Interactive mode
msgfy send -i
```

### Conversations

```bash
# List conversations
msgfy conv list
msgfy conv list --status open --channel ch_abc123

# Show conversation
msgfy conv show cv_abc123

# Show messages
msgfy conv messages cv_abc123 --limit 50

# Close conversation
msgfy conv close cv_abc123

# Reopen conversation
msgfy conv reopen cv_abc123

# Export conversation
msgfy conv export cv_abc123 > conversation.json
```

### Contacts

```bash
# List contacts
msgfy contact list
msgfy contact list --search "John" --tags vip

# Show contact
msgfy contact show ct_abc123

# Create contact
msgfy contact create \
  --name "John Doe" \
  --phone "+5544999999999" \
  --email "john@example.com" \
  --tags "vip,premium"

# Update contact
msgfy contact update ct_abc123 --set company="Acme Corp"

# Import from CSV
msgfy contact import contacts.csv --mapping "name:Nome,phone:Telefone"

# Export contacts
msgfy contact export --format csv > contacts.csv

# Merge duplicates
msgfy contact merge ct_abc123 ct_def456 --keep ct_abc123

# Delete contact
msgfy contact delete ct_abc123 --confirm
```

### Bots

```bash
# List bots
msgfy bot list

# Show bot details
msgfy bot show bt_abc123

# Create bot
msgfy bot create --name "Support Bot" --agent ag_xyz

# Start/Stop bot
msgfy bot start bt_abc123
msgfy bot stop bt_abc123

# Bot status
msgfy bot status bt_abc123

# View bot logs
msgfy bot logs bt_abc123 --follow
```

### Flows

```bash
# List flows
msgfy flow list
msgfy flow list --status published

# Show flow
msgfy flow show fl_abc123

# Execute flow manually
msgfy flow execute fl_abc123 --conversation cv_xyz

# Validate flow
msgfy flow validate fl_abc123

# Publish/Unpublish
msgfy flow publish fl_abc123
msgfy flow unpublish fl_abc123

# Export/Import
msgfy flow export fl_abc123 > flow.json
msgfy flow import flow.json --name "My Flow Copy"
```

### Knowledge Bases

```bash
# List knowledge bases
msgfy kb list

# Create knowledge base
msgfy kb create --name "Product Documentation"

# Show details
msgfy kb show kb_abc123

# Add document
msgfy kb doc add kb_abc123 --file manual.pdf --title "User Manual"
msgfy kb doc add kb_abc123 --url "https://docs.example.com" --title "API Docs"

# List documents
msgfy kb doc list kb_abc123

# Query knowledge base
msgfy kb query kb_abc123 "How do I reset my password?"

# Reprocess document
msgfy kb doc reprocess kb_abc123 doc_xyz

# Delete
msgfy kb delete kb_abc123 --confirm
```

### Webhooks

```bash
# List webhooks
msgfy webhook list

# Test webhook endpoint
msgfy webhook test https://example.com/webhook

# Simulate event
msgfy webhook simulate message.received --url https://example.com/webhook

# View recent events
msgfy webhook events --limit 20

# Start local listener (for debugging)
msgfy webhook listen --port 3000
```

### Configuration

```bash
# View config
msgfy config list

# Get/Set values
msgfy config get api_key
msgfy config set output_format json

# Switch profiles
msgfy config use staging

# Create profile
msgfy config profile create staging \
  --api-key lk_test_abc \
  --base-url https://staging-api.linktor.io
```

### Server (Self-Hosted)

```bash
# Start server
msgfy server start
msgfy server start --port 9000 --workers 4

# Server status
msgfy server status

# Run migrations
msgfy server migrate
msgfy server migrate --rollback 1

# Health check
msgfy server health

# Backup
msgfy server backup --output backup.tar.gz

# Plugin management
msgfy server plugin list
msgfy server plugin install whatsapp-unofficial
msgfy server plugin enable whatsapp-unofficial
msgfy server plugin disable whatsapp-unofficial
```

## Output Formats

```bash
# Table (default)
msgfy channel list

# JSON
msgfy channel list --output json

# IDs only (for scripts)
msgfy channel list --output ids
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Config file path |
| `--profile` | Use specific profile |
| `--output, -o` | Output format (table, json) |
| `--no-color` | Disable colored output |

## Examples

### Automation Script

```bash
#!/bin/bash

# Send promotional message to VIP contacts
msgfy contact list --tags vip --output ids | while read id; do
  contact=$(msgfy contact show $id --output json)
  phone=$(echo $contact | jq -r '.phone')
  msgfy send --channel ch_abc123 --to "$phone" --text "Special offer for you!"
  sleep 1
done
```

### CI/CD Integration

```yaml
# GitHub Actions
- name: Deploy Channel Config
  run: |
    msgfy auth login --api-key ${{ secrets.LINKTOR_API_KEY }}
    msgfy channel config ch_abc123 --set webhook_url=${{ env.WEBHOOK_URL }}
```

## Troubleshooting

### Authentication Issues

```bash
# Check current auth status
msgfy auth whoami

# Re-authenticate
msgfy auth logout
msgfy auth login
```

### Debug Mode

Set `MSGFY_DEBUG=1` to enable verbose logging:

```bash
MSGFY_DEBUG=1 msgfy channel list
```

## Contributing

See the main [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](../../LICENSE) for details.
