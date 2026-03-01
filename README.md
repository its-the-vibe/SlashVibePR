# SlashVibePR

A Slack Slash Command service that lets your team browse and share GitHub Pull Requests directly from Slack.

## How It Works

SlashVibePR sits on a Redis pub/sub bus alongside two companion services:

- **[slack-relay](https://github.com/its-the-vibe/slack-relay)** — forwards raw Slack events (slash commands, modal submissions, block actions) onto Redis channels.
- **[Poppit](https://github.com/its-the-vibe/poppit)** — executes `gh pr list` and publishes the output back to Redis.
- **[SlackLiner](https://github.com/its-the-vibe/slackliner)** — delivers formatted messages to Slack channels.

```
Slack ──► slack-relay ──► Redis ──► SlashVibePR ──► Redis ──► Poppit
                                         │                       │
                                         └──────────────────◄────┘
                                                   │
                                                   ▼
                                              SlackLiner ──► Slack
```

## Usage

In any Slack channel, type the `/pr` command:

| Command | Behaviour |
|---|---|
| `/pr` | Opens a repository chooser modal. Select a repo from the dropdown to see its open PRs. |
| `/pr <repo-name>` | Skips the repo chooser and loads open PRs for `<org>/<repo-name>` directly. |

**Examples:**

```
/pr
/pr my-service
/pr frontend-app
```

After selecting a PR from the list, SlashVibePR posts a formatted summary to the configured Slack channel.

## Installation & Setup

### Prerequisites

- Go 1.22+ (for local development)
- Docker & Docker Compose (for containerised deployment)
- A Redis instance accessible by all services
- A Slack App with a Bot Token (`xoxb-…`) and the `/pr` slash command configured
- The `gh` CLI available to Poppit (used to query GitHub PRs)

### 1. Clone the repository

```bash
git clone https://github.com/its-the-vibe/SlashVibePR.git
cd SlashVibePR
```

### 2. Configure the service

Copy the example files and edit them:

```bash
cp config.example.yaml config.yaml
cp .env.example .env
```

Edit **`config.yaml`** (non-secret settings — safe to commit):

```yaml
redis:
  addr: host.docker.internal:6379   # host:port of your Redis instance

channels:
  slash_commands: slack-commands
  view_submissions: slack-relay-view-submission
  block_actions: slack-relay-block-actions
  poppit_output: poppit:command-output

lists:
  poppit_commands: poppit:commands
  slackliner_messages: slack_messages

slack:
  channel_id: C0123456789    # Slack channel ID where PRs are posted

github:
  org: my-org                # GitHub organisation name

logging:
  level: INFO                # DEBUG | INFO | WARN | ERROR
```

Edit **`.env`** (secrets — **never** commit this file):

```dotenv
SLACK_BOT_TOKEN=xoxb-your-slack-bot-token-here
REDIS_PASSWORD=your-redis-password
```

### 3. Run with Docker Compose

```bash
docker compose up -d
```

The service reads `config.yaml` from `/etc/slashvibeprs/config.yaml` inside the container (mounted via the Compose file) and secrets from the `.env` file.

### 4. Run locally (without Docker)

```bash
go run .
```

Set `CONFIG_FILE` to point to an alternative config path if needed:

```bash
CONFIG_FILE=/path/to/config.yaml go run .
```

## Configuration Reference

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `SLACK_BOT_TOKEN` | **Yes** | Slack Bot OAuth token (`xoxb-…`) |
| `REDIS_PASSWORD` | No | Redis authentication password (leave empty if not set) |
| `CONFIG_FILE` | No | Path to the YAML config file (default: `config.yaml`) |

### config.yaml Fields

| Field | Default | Description |
|---|---|---|
| `redis.addr` | `host.docker.internal:6379` | Redis host and port |
| `channels.slash_commands` | `slack-commands` | Redis pub/sub channel for incoming `/pr` events |
| `channels.view_submissions` | `slack-relay-view-submission` | Redis channel for Slack modal submissions |
| `channels.block_actions` | `slack-relay-block-actions` | Redis channel for Slack block actions |
| `channels.poppit_output` | `poppit:command-output` | Redis channel for Poppit command results |
| `lists.poppit_commands` | `poppit:commands` | Redis list for outgoing Poppit tasks |
| `lists.slackliner_messages` | `slack_messages` | Redis list for outgoing SlackLiner messages |
| `slack.channel_id` | _(required)_ | Slack channel ID where PR summaries are posted |
| `github.org` | _(empty)_ | GitHub organisation prepended to the selected repository name |
| `logging.level` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, or `ERROR` |

## Development

### Run tests

```bash
go test ./...
```

### Build the binary

```bash
go build -o slashvibeprs .
```

### Build the Docker image

```bash
docker build -t slashvibeprs:latest .
```

## Contributing

Contributions are welcome! Please follow these steps:

1. **Fork** the repository and create a feature branch from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```
2. **Make your changes** and ensure the tests pass (`go test ./...`).
3. **Commit** using clear, descriptive commit messages.
4. **Open a Pull Request** against `main` and describe what your change does and why.

Please keep PRs focused and minimal. If you are proposing a significant change, open an issue first to discuss the approach.

### Related Issues

- [#9 — Skip repo chooser modal](https://github.com/its-the-vibe/SlashVibePR/issues/9)
- [#11 — Refactor repo selection modal](https://github.com/its-the-vibe/SlashVibePR/issues/11)
