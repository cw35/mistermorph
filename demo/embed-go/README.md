# Demo: Embed `mistermorph` as a Go library

This demo shows how another Go project can import `mistermorph/integration` and run in-process in three modes:

- `task`: run one-shot agent task.
- `telegram`: start a Telegram bot via `integration.NewTelegramBot(...)`.
- `slack`: start a Slack Socket Mode bot via `integration.NewSlackBot(...)`.

## Run: Task mode

From `demo/embed-go/`:

```bash
export OPENAI_API_KEY="..."
GOCACHE=/tmp/gocache GOPATH=/tmp/gopath GOMODCACHE=/tmp/gomodcache \
  go run . \
  --mode task \
  --task "List files in the current directory and summarize what this project is." \
  --model gpt-5.2
```

## Run: Telegram mode

```bash
export OPENAI_API_KEY="..."
export TG_BOT_TOKEN="123456:abcdef..."
GOCACHE=/tmp/gocache GOPATH=/tmp/gopath GOMODCACHE=/tmp/gomodcache \
  go run . \
  --mode telegram \
  --telegram-bot-token "$TG_BOT_TOKEN" \
  --telegram-group-trigger-mode smart \
  --telegram-max-concurrency 3
```

Optional allowlist example:

```bash
--telegram-allowed-chat-ids "12345,-100987654321"
```

## Run: Slack mode (Socket Mode)

```bash
export OPENAI_API_KEY="..."
export SLACK_BOT_TOKEN="xoxb-..."
export SLACK_APP_TOKEN="xapp-..."
GOCACHE=/tmp/gocache GOPATH=/tmp/gopath GOMODCACHE=/tmp/gomodcache \
  go run . \
  --mode slack \
  --slack-bot-token "$SLACK_BOT_TOKEN" \
  --slack-app-token "$SLACK_APP_TOKEN" \
  --slack-group-trigger-mode smart \
  --slack-max-concurrency 3
```

Optional allowlist example:

```bash
--slack-allowed-team-ids "T12345,T67890" --slack-allowed-channel-ids "C111,C222"
```

## Notes

- This demo uses OpenAI-compatible provider, so network access is required to actually run.
- `--inspect-prompt` and `--inspect-request` are supported in all modes.
- In `task` mode, the demo also registers example project tools (`list_dir`, `get_weather`) on top of selected built-ins.
- `telegram` and `slack` modes run until interrupted (`Ctrl+C`).
