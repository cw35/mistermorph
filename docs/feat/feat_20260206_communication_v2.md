---
date: 2026-02-06
title: MAEP v2 (Advanced Security & Reliability)
status: draft
---

# Feature: MAEP v2（非 MVP 增强）

## 1) Summary
本文件承接 `docs/feat/feat_20260206_maep.md`，定义非 MVP 的增强能力：
- 强 E2EE：X3DH + Double Ratchet。
- 可靠投递：ACK/retry/TTL/幂等去重。
- 离线与状态管理：prekeys、sessions、message queue。
- paid relay：x402 控制面。
- 群组能力：MLS（后续）。

术语继承：
- `peer_id`、`node_id`、`node_uuid` 的定义与 v1 保持一致。
- 协议层鉴权与路由继续以 `peer_id` 为准。

## 2) Goals
- 在不牺牲去中心拓扑前提下，提升安全属性（FS/PCS）。
- 支持异步建链路与更稳定的消息投递。
- 形成可商业化 relay 的标准计费扩展点。

## 3) Scope
本文件覆盖：
- X3DH 与 Double Ratchet 会话层。
- prekey 生命周期管理。
- 应用层 envelope 与投递语义增强。
- x402 relay 控制面。

不覆盖：
- 匿名网络（onion 多跳匿名）。
- 完整反审查网络设计。

## 4) X3DH + Double Ratchet

### 4.1 X3DH（异步建链路）
目标：对端不在线时仍可建立初始共享密钥。

参与密钥：
- 发起方 A：`IK_A`（身份密钥）, `EK_A`（一次性临时密钥）
- 接收方 B：`IK_B`（身份密钥）, `SPK_B`（signed prekey）, `OPK_B`（one-time prekey，可选）

流程：
1. A 从本地 contacts（out-of-band 导入）读取 B 的 prekey bundle：`IK_B + SPK_B + Sig(IK_B, SPK_B) + OPK_B`。
2. A 验证 `SPK_B` 签名归属 B 身份。
3. A 计算 X3DH 共享材料（3 或 4 个 DH 组合）并经 KDF 得到 `SK0`。
4. A 发送 `prekey_message`（含必要公钥材料与首条密文）。
5. B 重算 `SK0` 并将 `OPK_B` 标记为已消耗。

### 4.2 Double Ratchet（持续消息安全）
目标：实现前向保密（FS）与入侵后恢复（PCS）。

核心状态：
- `RootKey`
- `CKs`（发送链）、`CKr`（接收链）
- `MessageKey`（单消息密钥）
- DH ratchet 密钥对

流程（简化）：
1. 以 `SK0` 初始化 `RootKey/ChainKey`。
2. 发送时由 `CKs` 派生 `MessageKey`，随后推进 `CKs`。
3. 收到新 DH 公钥时触发 ratchet，更新 `RootKey` 并重置链。
4. 基于消息计数器处理乱序窗口并防重放。

## 5) Key Roles（v2）
| Key | 类型 | 生命周期 | 作用 |
|---|---|---|---|
| `IK` | Ed25519 | 长期 | 身份签名与绑定 `peer_id` |
| `SPK` | X25519 | 中期（7-30 天） | 异步建链路锚点 |
| `OPK` | X25519 | 一次性 | 强化首次会话抗重用能力 |
| `EK` | X25519 | 短期（单次） | 发起端临时建链路 |
| `RootKey` | 对称密钥 | 会话期滚动 | ratchet 根 |
| `CKs/CKr` | 对称密钥 | 每发/收推进 | 派生消息密钥 |
| `MK` | 对称密钥 | 单消息 | 实际加解密 payload |

关键要求：
- 私钥本地加密存储。
- `SPK/OPK` 库存监控与自动补充。
- 过期/吊销通过 `key_update` 广播。

## 6) Message & Reliability Enhancements

### 6.1 Envelope（建议）
```json
{
  "v": 2,
  "msg_id": "01JT...",
  "from_peer_id": "12D3KooW...",
  "to_peer_id": "12D3KooX...",
  "session_id": "s_...",
  "kind": "rpc_request",
  "created_at": "2026-02-06T12:00:00Z",
  "ttl_sec": 120,
  "ciphertext": "base64(...)",
  "sig": "ed25519(...)"
}
```

消息类型：
- `rpc_request`
- `rpc_response`
- `event`
- `ack`
- `key_update`

### 6.2 协议 ID（v2 固定）
- `hello`：`/maep/hello/2.0.0`
- `rpc`：`/maep/rpc/2.0.0`

### 6.3 `hello` 协商（v2 固定）
1. dialer 打开 `/maep/hello/2.0.0`，发送一个 JSON `hello` 后 half-close。
2. listener 读取后返回一个 JSON `hello` 并关闭流。
3. 双方计算 `negotiated_protocol`，结果不一致则断开并告警。
4. 未完成 `hello` 前不得处理 `/maep/rpc/2.0.0` 的业务请求。

### 6.4 Stream Framing（v2 固定）
v2 在 `/maep/rpc/2.0.0` 使用长度前缀帧：
- 每帧格式：`u32_be_len || frame_bytes`
- `frame_bytes` 为 UTF-8 JSON（`rpc_request/rpc_response/event/ack/key_update`）
- 单条 stream 允许多帧复用

默认限制：
- `max_frame_bytes = 1 MiB`
- `max_envelope_plaintext_bytes = 256 KiB`
- `max_inflight_streams_per_peer = 64`

### 6.5 投递语义
- at-least-once。
- ACK + retry window + TTL。
- 去重：`msg_id` 与 `rpc.id`。
- 超时后上报可观测状态，不静默失败。

默认参数：
- `ack_timeout = 5s`
- `retry_max = 5`
- `retry_backoff = 500ms..8s`（指数退避）

### 6.6 JSON 解析约束（继承 v1）
v2 沿用 v1 的 JSON/JCS 约束：
- 不允许 `null`
- 不允许浮点数（仅允许整数数值类型）
- 不允许重复 key
- 违反时使用 `ERR_INVALID_JSON_PROFILE`（`error.code=-32008`）

### 6.7 JSON-RPC 错误映射（v2 扩展）
v2 继承 v1 的错误码映射，并新增：

| Symbol | JSON-RPC `error.code` |
|---|---:|
| `ERR_SESSION_NOT_FOUND` | -32010 |
| `ERR_RATCHET_STATE_INVALID` | -32011 |
| `ERR_REPLAY_DETECTED` | -32012 |
| `ERR_PREKEY_EXHAUSTED` | -32013 |
| `ERR_PREKEY_INVALID` | -32014 |

规则：
- `error.code` 必须为整数。
- notification（无 `id`）不得返回响应。
- `error.message` 使用 `ERR_*` 符号码，细节放入 `error.data`。

## 7) Storage Extensions（v2）
在 v1 存储基础上新增：
- `prekeys`
  - `id`, `kind(spk/opk)`, `pub`, `priv_enc`, `status`, `expires_at`
- `sessions`
  - `session_id`, `peer_id`, `root_key_enc`, `cks_enc`, `ckr_enc`, `dh_state_enc`, `last_counter`
- `inbox_messages`
  - `msg_id`, `from_peer_id`, `ciphertext`, `received_at`, `ttl_sec`, `dedupe_state`
- `outbox_messages`
  - `msg_id`, `to_peer_id`, `ciphertext`, `retry_count`, `next_retry_at`, `ack_state`
- `relay_tokens`
  - `relay_id`, `token_enc`, `quota_bytes`, `expires_at`, `spent_bytes`

实现要求：
- session 更新必须事务化，防止 ratchet 状态错位。
- `msg_id + peer_id` 建唯一索引。

## 8) Paid Relay (x402)
计费仅在 relay 控制面，不进入数据面解密路径。

建议接口：
- `POST /relay/reservations`
- `POST /relay/quota/topup`

建议流程：
1. 客户端请求 reservation。
2. relay 返回 `402 Payment Required` 与支付要求。
3. 客户端提交支付证明（如 `PAYMENT-SIGNATURE`）。
4. relay 验证后签发短期 quota token。
5. 客户端在 relay 会话中携带 token。

## 9) Group Session (Future)
- 采用 MLS（RFC 9420）处理群组成员变更与群组密钥编排。
- 在 v2 后期评估并落地，不影响 v1/v2 点对点协议。

## 10) 为什么不直接采用完整 Signal 应用协议
1. 本项目目标是 agent RPC 网络，不是聊天产品协议。
2. 需要 libp2p + relay 的可插拔网络层与工程化 RPC 语义。
3. paid relay（x402）是控制面扩展，需独立定义。
4. 采用 Signal 的密码学核心（X3DH/Double Ratchet）即可满足安全要求。

## 11) Rollout Plan
### Phase A: Session Security
- [ ] 固化 `/maep/hello/2.0.0` 与 `/maep/rpc/2.0.0`。
- [ ] 固化 v2 长度前缀 framing（`u32_be_len || frame_bytes`）。
- [ ] 引入 X3DH + Double Ratchet。
- [ ] prekey 发布、轮换、吊销。
- [ ] replay/乱序窗口处理。

### Phase B: Reliability
- [ ] 固化 v2 错误码扩展映射（`-32010` 到 `-32014`）。
- [ ] ACK/retry/TTL。
- [ ] 去重索引与失败恢复。
- [ ] 基础离线消息队列。

### Phase C: Paid Relay
- [ ] relay 控制面 API。
- [ ] x402 支付挑战与 quota token。
- [ ] 配额计量（字节/消息/时长）。

### Phase D: Group
- [ ] MLS 群组会话。

## References
- JSON-RPC 2.0: https://www.jsonrpc.org/specification
- Signal X3DH: https://signal.org/docs/specifications/x3dh/
- Signal Double Ratchet: https://signal.org/docs/specifications/doubleratchet/
- MLS RFC 9420: https://www.ietf.org/rfc/rfc9420.html
- x402 Docs: https://docs.x402.org/
- x402 Coinbase Docs: https://docs.cdp.coinbase.com/x402/welcome
