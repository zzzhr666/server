# 接口文档

客户端使用 HTTP API `http://localhost:8080`，并连接原生 TCP 实时服务 `localhost:8081`。受保护 HTTP 请求使用 `Authorization: Bearer <token>`；除 `204 No Content` 外，HTTP 响应均为 JSON，错误格式为 `{"error":"error message"}`。

## HTTP

### 健康检查与认证

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/health` | 否 | 返回当前 logic-server 名称 |
| `POST` | `/auth/register` | 否 | 注册账号、创建玩家和 session，状态 `201` |
| `POST` | `/auth/login` | 否 | 登录并创建 session |
| `POST` | `/auth/logout` | 是 | 删除当前 session，状态 `204` |
| `GET` | `/auth/me` | 是 | 查询当前玩家 |

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

### 好友

| 方法 | 路径 | 请求体 | 成功结果 |
| --- | --- | --- | --- |
| `POST` | `/friends/requests` | `{"to_player_id":8}` | `204` |
| `GET` | `/friends/requests/incoming` | - | 收到的申请列表 |
| `GET` | `/friends/requests/outgoing` | - | 发出的申请列表 |
| `POST` | `/friends/requests/accept` | `{"from_player_id":7}` | `204` |
| `POST` | `/friends/requests/reject` | `{"from_player_id":7}` | `204` |
| `GET` | `/friends` | - | 好友及在线摘要 |
| `DELETE` | `/friends` | `{"friend_player_id":8}` | `204` |

### 局外成长

| 方法 | 路径 | 请求体 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/growth` | - | 返回等级与每项下一级信息 |
| `POST` | `/growth/upgrade` | `{"type":"attack"}` | 扣除金币并升级一项 |

合法 `type`：`attack`、`attack_speed`、`health`、`move_speed`。满级或金币不足返回 `409 Conflict`。

## TCP 实时协议

TCP 服务监听 `:8081`。每条应用帧使用 `GRTP` 格式：4 字节 ASCII magic `GRTP`、1 字节版本、4 字节大端 payload 长度、protobuf payload。版本和长度限制以 `internal/logic/realtime/frame.go` 为准；payload 是 `proto/realtime/v1/realtime.proto` 的 `ClientEnvelope` 或 `ServerEnvelope`。

客户端连接后第一帧必须是 `authenticate`，其中 `request_id` 必须非零：

```text
ClientEnvelope { request_id: 1, authenticate: { token: "session-token" } }
ServerEnvelope { request_id: 1, authenticated: { player_id: 7 } }
```

认证成功后服务端自动尝试恢复活跃对局。自动推送和好友事件的 `request_id` 为 `0`；客户端请求的响应沿用其非零 `request_id`。

| 客户端 payload | 行为 |
| --- | --- |
| `heartbeat` | 刷新在线状态，返回 `heartbeat_ack` |
| `match_start { weapon }` | 开始匹配，合法武器为 `sword`、`dagger`、`axe`、`bow`；空值默认 `sword` |
| `match_cancel` | 取消等待中的匹配，返回 `match_canceled` |
| `match_resume` | 请求当前玩家的活跃对局 |

匹配和恢复使用 `match_result`，其中包含 `status`、`room_name`、`token`、`battle_node_name` 和 `battle_udp_addr`。`status=matched` 后，客户端向 `battle_udp_addr` 发送 UDP `ClientHello(room_name, player_id, token)`。

业务请求失败时服务端发送 `error { code, message }` 并保持连接；协议帧、认证和写入失败会关闭连接。主动事件包括 `connection_replaced`、`friend_presence_changed`、`friend_removed`、`friend_request_received`、`friend_request_handled` 和 `match_result`。

## UDP 战斗协议

UDP payload 使用 `proto/battle/v1/session.proto` 的 `ClientPacket` 与 `ServerPacket`。

1. 收到 TCP `matched` 后，向 `battle_udp_addr` 发送 `ClientHello(room_name, player_id, token)`。
2. 收到 `ServerHello` 后保存服务器分配的 conversation，开始接收快照。
3. 每 5 秒发送 `ClientHeartbeat`，并按输入状态发送 `ClientInput`。
4. 在 `BATTLE_PHASE_REWARD_SELECTION` 时，使用 snapshot 中的 `current_options` 发送 `ChooseBlessing`。

默认超时：有效 UDP 输入或心跳缺失 15 秒时 session 断开；房间内所有 session 都断开 90 秒后，服务器以 `all_players_disconnected` 结束房间。完整字段和枚举以 [session.proto](../proto/battle/v1/session.proto) 为准。
