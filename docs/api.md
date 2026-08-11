# 接口文档

客户端使用 HTTP `http://localhost:8080` 完成健康检查、注册和登录，并连接原生 TCP 实时服务 `localhost:8081` 完成其余大厅操作。HTTP 响应为 JSON，错误格式为 `{"error":"error message"}`。

## HTTP

### 健康检查与认证

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/health` | 否 | 返回当前 logic-server 名称 |
| `POST` | `/auth/register` | 否 | 注册账号、创建玩家和 session，状态 `201` |
| `POST` | `/auth/login` | 否 | 登录并创建 session |

注册请求：

```json
{
  "username":"alice",
  "password":"password123",
  "nickname":"Alice",
  "avatar":"alice.png",
  "email":"alice@example.com",
  "phone":"13800000000"
}
```

注册与登录成功响应：

```json
{
  "token":"session-token",
  "player":{"id":1,"nickname":"Alice","avatar":"alice.png","email":"alice@example.com","phone":"13800000000","coins":0}
}
```

## TCP 实时协议

TCP 服务监听 `:8081`。每条应用帧使用 `GRTP` 格式：4 字节 ASCII magic `GRTP`、1 字节版本 `1`、4 字节大端 payload 长度、4 字节大端 CRC32C（Castagnoli）和 protobuf payload。payload 长度必须大于 0 且不超过 1 MiB；payload 是 `proto/realtime/v1/realtime.proto` 的 `ClientEnvelope` 或 `ServerEnvelope`。

客户端连接后第一帧必须是 `authenticate`，其中 `request_id` 必须非零：

```text
ClientEnvelope { request_id: 1, authenticate: { token: "session-token" } }
ServerEnvelope { request_id: 1, authenticated: { player_id: 7 } }
```

认证成功后服务端自动尝试恢复活跃对局。所有客户端请求都必须携带非零 `request_id` 和一个 payload；客户端请求的响应沿用对应 `request_id`。自动恢复结果、匹配结果和好友事件等服务端主动推送使用 `request_id=0`。

| 客户端 payload | 成功响应 | 行为 |
| --- | --- | --- |
| `heartbeat` | `heartbeat_ack` | 刷新在线状态 |
| `match_start { weapon }` | `match_result` | 开始匹配，合法武器为 `sword`、`dagger`、`axe`、`bow`；空值默认 `sword` |
| `match_cancel` | `match_canceled` | 取消等待中的匹配 |
| `match_resume` | `match_result` | 请求当前玩家的活跃对局 |
| `player_get` | `player` | 返回当前玩家档案与金币 |
| `growth_get` | `growth` | 返回成长等级和每项升级选项 |
| `growth_upgrade { type }` | `growth_upgrade_result` | 扣除金币并升级，返回最新成长、余额和费用 |
| `friend_request_send { to_player_id }` | `friend_request_sent` | 发送好友申请 |
| `friend_request_list_incoming` | `friend_requests` | 返回收到的好友申请 |
| `friend_request_list_outgoing` | `friend_requests` | 返回发出的好友申请 |
| `friend_request_accept { from_player_id }` | `friend_request_handled_ack` | 接受好友申请 |
| `friend_request_reject { from_player_id }` | `friend_request_handled_ack` | 拒绝好友申请 |
| `friend_list` | `friends` | 返回好友档案与在线状态摘要 |
| `friend_delete { friend_player_id }` | `friend_deleted` | 删除好友关系 |
| `logout` | `logout_ack` | 删除当前 session，服务端随后关闭 TCP 连接 |

匹配和恢复使用 `match_result`，其中包含 `status`、`room_name`、`token`、`battle_node_name` 和 `battle_udp_addr`。`status=matched` 后，客户端向 `battle_udp_addr` 发送 UDP `ClientHello(room_name, player_id, token)`。

成长升级的合法 `type` 为 `attack`、`attack_speed`、`health`、`move_speed`。业务请求失败时服务端发送 `error { code, message }` 并保持连接；`code` 取值为 `UNAUTHENTICATED`、`INVALID_ARGUMENT`、`NOT_FOUND`、`CONFLICT` 或 `INTERNAL`。协议帧错误、认证失败、logout 和写入失败会关闭连接。主动事件包括 `connection_replaced`、`friend_presence_changed`、`friend_removed`、`friend_request_received`、`friend_request_handled` 和 `match_result`。

## UDP 战斗协议

UDP payload 使用 `proto/battle/v1/session.proto` 的 `ClientPacket` 与 `ServerPacket`。

1. 收到 TCP `matched` 后，向 `battle_udp_addr` 发送 `ClientHello(room_name, player_id, token)`。
2. 收到 `ServerHello` 后保存服务器分配的 conversation，开始接收快照。
3. 每 5 秒发送 `ClientHeartbeat`，并按输入状态发送 `ClientInput`。
4. 在 `BATTLE_PHASE_REWARD_SELECTION` 时，使用 snapshot 中的 `current_options` 发送 `ChooseBlessing`。

默认超时：有效 UDP 输入或心跳缺失 15 秒时 session 断开；房间内所有 session 都断开 90 秒后，服务器以 `all_players_disconnected` 结束房间。完整字段和枚举以 [session.proto](../proto/battle/v1/session.proto) 为准。
