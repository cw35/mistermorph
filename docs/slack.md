# Slack Setup (Socket Mode)

本文说明如何为 `mistermorph slack` 准备凭据，尤其是“只有 `client_id/client_secret`”时如何获取可用 token。

## 1. 三类凭据的区别

- `client_id` / `client_secret`
  - 用于 OAuth 换票（`code -> token`）。
  - 不能直接拿来调用 `mistermorph slack`。
- Bot Token（`xoxb-...`）
  - 用于调用 Web API（如 `chat.postMessage`）。
  - `mistermorph slack` 必填：`slack.bot_token`。
- App Token（`xapp-...`）
  - 用于 Socket Mode 建立 WebSocket 连接（`apps.connections.open`）。
  - `mistermorph slack` 必填：`slack.app_token`。

## 2. 先开 Socket Mode

在 Slack App 管理后台：

1. 进入 `Socket Mode`。
2. 打开 `Enable Socket Mode`。

## 3. 获取 Bot Token (`xoxb-...`)

### 方式 A：后台直接安装（推荐）

1. 进入 `OAuth & Permissions`。
2. 在 `Bot Token Scopes` 添加最小必要 scopes（见下节）。
3. 点击 `Install to Workspace`（如果已安装且 scope 有变更，点 `Reinstall`）。
4. 复制 `Bot User OAuth Token`（`xoxb-...`）。

### 方式 B：只有 `client_id/client_secret` 时（OAuth 换票）

先完成 OAuth 授权拿到 `code`，然后调用：

```bash
curl -X POST https://slack.com/api/oauth.v2.access \
  -d client_id=YOUR_CLIENT_ID \
  -d client_secret=YOUR_CLIENT_SECRET \
  -d code=AUTH_CODE \
  -d redirect_uri=YOUR_REDIRECT_URI
```

返回 JSON 中的 `access_token`（通常是 `xoxb-...`）就是 bot token。

## 4. 获取 App Token (`xapp-...`)

`xapp` 不能通过 `client_id/client_secret` 的 OAuth 换票获得，需要在后台手动生成：

1. 进入 `Basic Information`。
2. 找到 `App-Level Tokens`。
3. 点击 `Generate Token and Scopes`。
4. 添加 scope：`connections:write`。
5. 生成后复制 `xapp-...`。

## 5. 建议 scopes（Phase A）

本仓库当前 Slack Phase A（Socket Mode + 文本收发）建议配置：

- `app_mentions:read`
- `channels:history`
- `groups:history`
- `im:history`
- `mpim:history`
- `chat:write`

## 6. 写入配置

可用环境变量（推荐）：

```bash
export MISTER_MORPH_SLACK_BOT_TOKEN='xoxb-...'
export MISTER_MORPH_SLACK_APP_TOKEN='xapp-...'
```

或写到配置文件：

```yaml
slack:
  bot_token: "xoxb-..."
  app_token: "xapp-..."
  allowed_team_ids: []
  allowed_channel_ids: []
  group_trigger_mode: "smart" # strict|smart|talkative
  addressing_confidence_threshold: 0.6
  addressing_interject_threshold: 0.6
  task_timeout: "0s"
  max_concurrency: 3
```

## 7. 启动示例

```bash
go run ./cmd/mistermorph slack \
  --slack-bot-token "$MISTER_MORPH_SLACK_BOT_TOKEN" \
  --slack-app-token "$MISTER_MORPH_SLACK_APP_TOKEN"
```

## 8. 常见报错

- `missing slack.bot_token` / `missing slack.app_token`
  - 没有传 token，或环境变量名不对。
- `slack auth.test failed: invalid_auth`
  - `xoxb` 无效、过期、复制错误，或装错 workspace。
- `slack apps.connections.open failed: not_allowed_token_type`
  - 传了非 `xapp` token，或 `xapp` 没有 `connections:write`。
- 收不到群消息
  - 检查 bot 是否在目标频道、scope 是否齐全、频道/team allowlist 是否拦截。

## 9. 安全建议

- 不要把 `xoxb`/`xapp` 提交到仓库。
- 线上优先用环境变量或 secret manager 注入。
- 日志中避免打印完整 token。
