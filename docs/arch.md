# MisterMorph Architecture

## 1. System Architecture

```text
                           +-------------------------+
                           |      User Surface       |
                           | CLI / Telegram / Slack  |
                           +------------+------------+
                                        |
                   +--------------------+--------------------+
                   |                                         |
         +---------v---------+                     +---------v---------+
         | CLI bootstrap     |                     | integration API   |
         | cmd/mistermorph/* |                     | integration/*     |
         +---------+---------+                     +---------+---------+
                   |                                         |
                   +--------------------+--------------------+
                                        |
                           +------------v------------+
                           | Runtime Assembly Layer  |
                           | config snapshot + deps  |
                           +------------+------------+
                                        |
      +-----------------+---------------+-----------------------+
      |                 |                                       |
 +----v-----+   +-------v--------+                     +--------v--------+
 | One-shot |   | Channel runtime|                     | Heartbeat       |
 | runtime  |   | telegram/slack |                     | scheduler       |
 | run/serve|   | event workers  |                     | periodic checks |
 +----+-----+   +-------+--------+                     +--------+--------+
      |                 |                                       |
      +-----------------+-------------------+-------------------+
                                        |
                               +--------v--------+
                               |   agent.Engine  |
                               +---+---------+---+
                                   |         |
                          +--------v--+   +--v--------+
                          | llm.Client|   | tools.Reg |
                          +-----+-----+   +-----+-----+
                                |               |
                          +-----v-----+   +-----v------------------+
                          | providers |   | builtin/tools/adapters |
                          +-----------+   +------------------------+
Cross-cutting: guard, skills/prompt blocks, inspect dump, bus idempotency, file_state_dir, HEARTBEAT.md
```

## 2. Main Execution Flow

```text
task/event
  -> build prompt/messages/meta
  -> agent.Engine.Run
     -> step loop
        -> LLM call
        -> parse (plan | tool_call | final)
        -> optional tool execute
        -> update plan/history/metrics
     -> final output (guard redact if needed)
```

This flow is implemented by `agent/engine.go` and `agent/engine_loop.go` and is shared across all entrypoints.

## 3. Two Runtime Families

### 3.1 One-shot (`run` / `serve`)

```text
CLI command -> config/registry/guard setup -> agent.Engine.Run -> output/json
```

- Entrypoints: `cmd/mistermorph/runcmd/run.go`, `cmd/mistermorph/daemoncmd/serve.go`
- Characteristics: single task execution or queued execution; no platform event consumer loop

### 3.2 Channel (Telegram / Slack)

```text
platform event
  -> inbound adapter
  -> inproc bus
  -> per-conversation worker (serial)
  -> run*Task -> agent.Engine
  -> outbound publish
  -> delivery adapter
  -> platform send
```

- Telegram: `internal/channelruntime/telegram/*`
- Slack: `internal/channelruntime/slack/*`

## 4. Existing Topic Docs (Links Only)

The following areas already have formal docs, so this file only links them:

- Prompt system: [`./prompt.md`](./prompt.md)
- Tools system: [`./tools.md`](./tools.md)
- Security / Guard: [`./security.md`](./security.md)
- Skills system: [`./skills.md`](./skills.md)
- Heartbeat feature notes: [`./feat/feat_20260204_heartbeat.md`](./feat/feat_20260204_heartbeat.md)
- Telegram runtime behavior: [`./telegram.md`](./telegram.md)
- Slack Socket Mode: [`./slack.md`](./slack.md)
- Bus design and implementation: [`./bus.md`](./bus.md), [`./bus_impl.md`](./bus_impl.md)

## 5. Key Areas Without Standalone Docs

### 5.1 Integration Embedding Layer

```text
host app
  -> integration.DefaultConfig + Set(...)
  -> rt := integration.New(cfg)      // snapshot at init
  -> rt.RunTask(...)                 // one-shot
  -> rt.NewTelegramBot/NewSlackBot   // long-running
```

Notes:

- `integration` is the third-party reuse entrypoint; host apps do not need to depend on CLI command wiring.
- `integration.New(cfg)` builds a snapshot of effective runtime config at initialization time.
- Code: `integration/runtime.go`, `integration/runtime_snapshot*.go`, `integration/channel_bots.go`

### 5.2 Memory Status

```text
telegram private user
  -> resolve identity (ext:telegram:<user_id>)
  -> load/inject summaries
  -> update short-term + long-term markdown
```

Notes:

- Runtime-level memory integration is currently wired for Telegram only; Slack memory integration is not yet wired.
- Storage model lives in `memory/*`, runtime integration is in `internal/channelruntime/telegram/runtime_task.go`.

### 5.3 Heartbeat Runtime Path

```text
heartbeat ticker (runtime scheduler)
  -> heartbeatutil.Tick(state, buildTask, enqueueTask)
  -> BuildHeartbeatTask(HEARTBEAT.md)
  -> enqueue heartbeat job (meta.trigger=heartbeat)
  -> agent.Engine.Run (normal tools/skills enabled)
  -> summary output (runtime-defined sink, e.g. logs/chat)
```

Notes:

- Heartbeat shares the same agent execution core; it differs mainly by scheduler path and metadata envelope.
- Scheduler-side skip reasons include `already_running`, `worker_busy`, `worker_queue_full`, and `empty_task`.
- Consecutive failures are tracked by `heartbeatutil.State`; alert escalation is emitted after threshold.
- Code:
  - shared helpers: `internal/heartbeatutil/heartbeat.go`, `internal/heartbeatutil/scheduler.go`
  - runtime integrations: `cmd/mistermorph/daemoncmd/serve.go`, `internal/channelruntime/telegram/runtime.go`

### 5.4 Plan Creation and Progress Lifecycle

```text
registry build
  -> register plan_create (if enabled)
  -> prompt block hints "use plan_create for multi-step tasks"
  -> agent loop
     -> plan source A: model returns type="plan"
     -> plan source B: model calls plan_create tool
        -> plan_create sub-LLM call -> JSON plan
     -> NormalizePlanSteps
     -> each successful non-plan_create tool call:
        AdvancePlanOnSuccess (in_progress -> completed, next pending -> in_progress)
     -> optional plan progress publish (Telegram hook)
     -> final: CompleteAllPlanSteps
```

Plan sources and wiring:

- `plan_create` registration happens during runtime assembly when the runtime enables plan tooling (`PlanTool` feature) and calls `RegisterPlanTool(...)`.
- `RegisterPlanTool(...)` effectively registers `plan_create` only when:
  - registry is non-nil,
  - config switch `tools.plan_create.enabled` is true (default true).
- `tools.plan_create.max_steps` controls default plan step cap (fallback default is 6).
- Prompt guidance is injected by `AppendPlanCreateGuidanceBlock(...)` only if `plan_create` exists in the registry.
- In current flow, plan generation is expected to come from `plan_create`; engine also keeps compatibility for direct `type="plan"` responses.

Plan normalization and advancement invariants:

- `agent.NormalizePlanSteps`:
  - trims step text,
  - drops empty steps,
  - normalizes status to `pending|in_progress|completed`,
  - enforces at most one `in_progress` (promotes first pending when needed).
- Normalization is applied at plan creation time; advancement assumes the stored plan is already normalized.
- `agent.AdvancePlanOnSuccess` now returns `ok=false` when no valid `in_progress` step exists (prevents phantom "step 0 completed").
- On final output, remaining non-completed steps are marked completed (`CompleteAllPlanSteps`).

`plan_create` strictness:

- `Execute(...)` reuses `NormalizePlanSteps(...)` before cap.
- If normalized steps are empty, `Execute(...)` returns `invalid plan_create response: empty steps`.
- Otherwise, `max_steps` cap is applied.
- The tool no longer auto-fills placeholder names for empty steps.

Telegram progress updates:

- Telegram runtime installs `WithPlanStepUpdate(...)` during task-run engine construction.
- On each completed plan step, it renders a short progress message via `generateTelegramPlanProgressMessage(...)` and publishes outbound with correlation id `telegram:plan:<chat_id>:<message_id>`.
- Progress send is skipped when plan is missing/empty or `CompletedIndex < 0`.

## 6. State Directory and Naming Baseline

```text
file_state_dir (default ~/.morph)
├── HEARTBEAT.md
├── contacts/
│   ├── ACTIVE.md
│   ├── INACTIVE.md
│   ├── bus_inbox.json
│   └── bus_outbox.json
├── memory/
│   ├── index.md
│   └── YYYY-MM-DD/<sanitized-session-id>.md
├── guard/
│   ├── audit/guard_audit.jsonl
│   └── approvals/guard_approvals.json
└── skills/<skill>/SKILL.md
```

Additional notes:

- `HEARTBEAT.md` is the default heartbeat checklist input (`statepaths.HeartbeatChecklistPath()`).
- Memory short-term filenames come from sanitized `session_id` values (letters, digits, `-`, `_`).
- Contacts bus dedupe keys:
  - inbox: `(channel, platform_message_id)`
  - outbox: `(channel, idempotency_key)`

## 7. Code Navigation

Recommended reading order:

1. `cmd/mistermorph/root.go` (entrypoint assembly)
2. `integration/runtime.go` (embedding entrypoint)
3. `agent/engine.go` + `agent/engine_loop.go` (execution core)
4. `internal/channelruntime/telegram/runtime.go`, `internal/channelruntime/telegram/runtime_task.go`, and `internal/channelruntime/slack/runtime.go` (channel flow)
5. `internal/bus/*` and `internal/bus/adapters/*` (message bus and adapters)
