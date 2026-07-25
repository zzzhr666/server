# API

默认 HTTP Base URL：

```text
http://localhost:8081
```

如果通过 nginx 启动，也可以使用：

```text
http://localhost:8080
```

HTTP 登录态使用：

```text
Authorization: Bearer <token>
```

WebSocket 登录态使用：

```text
token: <token>
```

错误响应统一为：

```json
{"error":"error message"}
```

## HTTP

### `GET /health`

健康检查。

```bash
curl http://localhost:8081/health
```

成功响应：

```text
ok server_name = logic-1
```

### `POST /auth/register`

注册账号，同时创建玩家资料和 session。

请求：

```json
{
  "username": "alice",
  "password": "password123",
  "nickname": "Alice",
  "avatar": "alice.png",
  "email": "alice@example.com",
  "phone": "13800000000"
}
```

成功状态码：`201 Created`

响应：

```json
{
  "token": "session-token",
  "player": {
    "id": 1,
    "nickname": "Alice",
    "avatar": "alice.png",
    "email": "alice@example.com",
    "phone": "13800000000"
  }
}
```

### `POST /auth/login`

登录并创建 session。

请求：

```json
{
  "username": "alice",
  "password": "password123"
}
```

成功状态码：`200 OK`

响应：

```json
{
  "token": "session-token",
  "player": {
    "id": 1,
    "nickname": "Alice",
    "avatar": "alice.png",
    "email": "alice@example.com",
    "phone": "13800000000"
  }
}
```

### `GET /auth/me`

查询当前 token 对应的玩家资料。

请求头：

```text
Authorization: Bearer <token>
```

成功状态码：`200 OK`

响应：

```json
{
  "id": 1,
  "nickname": "Alice",
  "avatar": "alice.png",
  "email": "alice@example.com",
  "phone": "13800000000"
}
```

### `POST /auth/logout`

删除当前 session。

请求头：

```text
Authorization: Bearer <token>
```

成功状态码：`204 No Content`

### `POST /friends/requests`

发送好友请求。

请求头：

```text
Authorization: Bearer <token>
```

请求：

```json
{"to_player_id":8}
```

成功状态码：`204 No Content`

### `GET /friends/requests/incoming`

查看收到的好友请求。

请求头：

```text
Authorization: Bearer <token>
```

响应：

```json
{
  "requests": [
    {
      "from_player_id": 7,
      "to_player_id": 8,
      "created_at": "2026-07-25T10:00:00Z"
    }
  ]
}
```

### `GET /friends/requests/outgoing`

查看发出的好友请求。

请求头：

```text
Authorization: Bearer <token>
```

响应格式同 `/friends/requests/incoming`。

### `POST /friends/requests/accept`

接受好友请求。

请求头：

```text
Authorization: Bearer <token>
```

请求：

```json
{"from_player_id":7}
```

成功状态码：`204 No Content`

### `POST /friends/requests/reject`

拒绝好友请求。

请求头：

```text
Authorization: Bearer <token>
```

请求：

```json
{"from_player_id":7}
```

成功状态码：`204 No Content`

### `GET /friends`

查看好友列表。

请求头：

```text
Authorization: Bearer <token>
```

响应：

```json
{
  "friends": [
    {
      "player_id": 8,
      "nickname": "Bob",
      "avatar": "bob.png",
      "online": true,
      "status": "online",
      "updated_at": "2026-07-25T10:00:00Z"
    }
  ]
}
```

### `DELETE /friends`

删除好友。

请求头：

```text
Authorization: Bearer <token>
```

请求：

```json
{"friend_player_id":8}
```

成功状态码：`204 No Content`

## WebSocket

连接：

```text
GET ws://localhost:8081/ws
token: <token>
```

客户端消息：

| type | 说明 |
| --- | --- |
| `heartbeat` | 刷新在线状态 |
| `match_start` | 开始匹配 |
| `match_cancel` | 取消匹配 |

示例：

```json
{"type":"heartbeat"}
```

```json
{"type":"match_start"}
```

服务端消息：

```json
{"type":"match_result","status":"waiting"}
```

```json
{
  "type": "match_result",
  "status": "matched",
  "room_name": "room-1001-1002",
  "token": "room-token",
  "battle_node_name": "battle-demo",
  "battle_kcp_addr": "127.0.0.1:7001"
}
```

```json
{"type":"match_error","error":"no available BattleNode"}
```

```json
{"type":"match_canceled"}
```

好友和连接事件：

```json
{"type":"friend_presence_changed","player_id":8,"online":true,"status":"online"}
```

```json
{"type":"friend_request_received","player_id":8}
```

```json
{"type":"friend_request_handled","player_id":8}
```

```json
{"type":"friend_removed","player_id":8}
```

```json
{"type":"connection_replaced"}
```

## Battle UDP

UDP 地址来自 WebSocket `match_result.battle_kcp_addr`。当前包体是 protobuf：

```text
proto/battle/v1/session.proto
```

客户端发送 `battle.v1.ClientPacket`。

`hello`：

```text
room_name: string
player_id: int64
token: string
```

`move_input`：

```text
room_name: string
player_id: int64
x: float
y: float
```

服务端返回 `battle.v1.ServerPacket`。

| payload | 字段 |
| --- | --- |
| `hello` | `conv`, `message` |
| `game_start` | `room_name`, `player_ids` |
| `snapshot` | `room_name`, `entities` |
| `game_over` | `room_name`, `player_ids`, `reason` |
| `error` | `code`, `message` |

`snapshot.entities`：

```text
entity: uint32
x_position: float
y_position: float
x_direction: float
y_direction: float
current_health: int32
max_health: int32
```

调试命令：

```bash
go run ./tools/battle_udp_client \
  -addr 127.0.0.1:7001 \
  -room <room_name> \
  -token <room_token> \
  -player <player_id> \
  -move-x 1 \
  -move-y 0
```

## gRPC

gRPC 接口主要供服务间调用，也可以用工具手动调试。

| 服务 | 默认地址 | proto | 常用方法 |
| --- | --- | --- | --- |
| `state.v1.StateService` | `127.0.0.1:9001` | `proto/state/v1/state.proto` | 账号、玩家、session、在线状态、好友、实时事件 |
| `rcenter.v1.RCenterService` | `127.0.0.1:9002` | `proto/rcenter/v1/rcenter.proto` | `RegisterBattleNode`, `ListBattleNodes`, `StartMatch`, `CancelMatch` |
| `battle.v1.BattleControlService` | `127.0.0.1:9101` | `proto/battle/v1/battle.proto` | `CreateRoom`, `JoinRoom`, `EndRoom` |

创建 battle room：

```bash
go run ./tools/create_battle_room -room room-1 -token token-1 -players 1001,1002
```

结束 battle room：

```bash
go run ./tools/end_battle_room -room room-1 -reason manual_end
```
