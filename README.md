# ListHaul

ListHaul is a client-side IMAP inbox cleanup tool. It polls your mailbox on a schedule, evaluates Sieve-inspired filter rules against message headers, and applies actions that many providers do not expose in their UI.

## Features

- IMAP polling with configurable interval and lookback window
- TOML configuration with Sieve-style tests (`header`, `exists`, `not`, `anyof`, `allof`)
- Configurable actions: `fileinto`, `discard`, `mark_read`, `flag`, `stop`
- Optional UID state tracking to avoid re-processing messages in the lookback window
- Dry-run mode for safe testing

## Requirements

- [mise](https://mise.jdx.dev/) for Go toolchain and tasks
- An IMAP account with app password or dedicated credentials

## Quick start

```bash
mise trust
mise install
cp config.example.toml listhaul.toml
export LISTHAUL_IMAP_PASSWORD='your-app-password'
mise run validate
mise run build
./bin/listhaul run-once -c listhaul.toml
```

## Commands

| Command | Description |
|---------|-------------|
| `listhaul run -c listhaul.toml` | Long-running poll loop |
| `listhaul run-once -c listhaul.toml` | Single poll cycle |
| `listhaul validate -c listhaul.toml` | Validate config without IMAP |
| `listhaul dry-run -c listhaul.toml` | Evaluate rules, log actions, no mutations |

Use `-v debug` for verbose logging.

## Configuration

See [`config.example.toml`](config.example.toml) for a full annotated example.

### Poll settings

```toml
[poll]
interval = "10m"    # how often to check mail
lookback = "60m"    # only consider mail from this window
unseen_only = false # optionally limit to UNSEEN messages
```

### Account

```toml
[account]
host = "imap.example.com"
port = 993
username = "user@example.com"
password_env = "LISTHAUL_IMAP_PASSWORD"
mailbox = "INBOX"
```

Store passwords in environment variables referenced by `password_env`. Do not commit credentials.

### Actions

Only actions listed under `[actions].enabled` may appear in rules:

```toml
[actions]
enabled = ["fileinto", "discard", "mark_read", "flag", "stop"]
```

| Action | Fields | Behavior |
|--------|--------|----------|
| `fileinto` | `folder` | Move message to folder (uses IMAP MOVE when available) |
| `discard` | — | Mark deleted and expunge |
| `mark_read` | — | Set `\Seen` flag |
| `flag` | `flag`, `set` | Add or remove an IMAP flag |
| `stop` | — | Stop evaluating further rules for this message |

### Rules

Rules are evaluated top-to-bottom. The first matching rule wins unless it includes `stop` after other actions in the same rule.

```toml
[[rules]]
name = "newsletters"

[rules.if]
header = { names = ["List-Id"], match = "contains", values = ["lists.example.com"] }

[[rules.then]]
action = "fileinto"
folder = "INBOX/Newsletters"

[[rules.then]]
action = "stop"
```

#### Test types

| Test | Example |
|------|---------|
| `header` | `header = { names = ["Subject"], match = "matches", values = ["*sale*"] }` |
| `exists` | `exists = ["List-Unsubscribe"]` |
| `not` | `not = { exists = ["List-Id"] }` |
| `anyof` | `anyof = [ { ... }, { ... } ]` |
| `allof` | `allof = [ { ... }, { ... } ]` |

Match types for `header`: `is`, `contains`, `matches` (glob via `*` and `?`).

### State tracking

When enabled, ListHaul records processed `(mailbox, uid)` pairs to avoid re-applying rules within the overlapping lookback window:

```toml
[state]
enabled = true
path = "~/.listhaul/state.json"
```

## Development

```bash
mise run test
mise run fmt
mise run lint   # requires golangci-lint
mise run build
```

## Docker

Run one container per mailbox with Docker Compose. Each service mounts its own config and keeps UID state in a dedicated volume.

```bash
cp .env.example .env
mkdir -p config
cp docker/mailbox.example.toml config/gmail.toml
cp docker/mailbox.example.toml config/yahoo.toml
# Edit each file: IMAP host, username, password_env, rules.
# Use path = "/data/state.json" for state (already set in the example).

docker compose up -d --build
docker compose logs -f gmail
```

One-off commands:

```bash
docker compose run --rm gmail validate -c /config/listhaul.toml
docker compose run --rm gmail dry-run -c /config/listhaul.toml
```

To add another mailbox, copy a service block in `docker-compose.yml`, point it at a new config file, add a named volume, and set the password in `.env`.

## Running as a service

### systemd (Linux)

```ini
[Unit]
Description=ListHaul IMAP filter
After=network-online.target

[Service]
Type=simple
Environment=LISTHAUL_IMAP_PASSWORD=...
ExecStart=/usr/local/bin/listhaul run -c /etc/listhaul/listhaul.toml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### launchd (macOS)

Create `~/Library/LaunchAgents/com.listhaul.agent.plist` pointing at your binary and config, with `LISTHAUL_IMAP_PASSWORD` set via `EnvironmentVariables`.

## License

MIT
