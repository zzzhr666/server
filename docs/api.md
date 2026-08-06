# API

默认直连地址是 `http://localhost:8081`；启用 nginx 时使用 `http://localhost:8080`。

受保护 HTTP 请求使用 `Authorization: Bearer <token>`。WebSocket 通过 `GET /ws` 建立，并传入请求头 `token: <token>`。除 `204 No Content` 外，响应均为 JSON；错误格式为：

```json
{"error":"error message"}
```

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

登录请求只包含 `username` 和 `password`。注册与登录成功响应：

```json
{
  "token":"session-token",
  "player":{
    "id":1,
    "nickname":"Alice",
    "avatar":"alice.png",
    "email":"alice@example.com",
    "phone":"13800000000",
    "coins":0
  }
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

申请列表：

```json
{"requests":[{"from_player_id":7,"to_player_id":8,"created_at":"2026-08-05T10:00:00Z"}]}
```

好友列表：

```json
{
  "friends":[
    {"player_id":8,"nickname":"Bob","avatar":"bob.png","online":true,"status":"online","updated_at":"2026-08-05T10:00:00Z"}
  ]
}
```

### 局外成长

| 方法 | 路径 | 请求体 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/growth` | - | 返回等级与每项下一级信息 |
| `POST` | `/growth/upgrade` | `{"type":"attack"}` | 扣除金币并升级一项 |

合法 `type`：`attack`、`attack_speed`、`health`、`move_speed`。满级或金币不足返回 `409 Conflict`。

```json
{
  "player_id":7,
  "attack_level":2,
  "attack_speed_level":1,
  "health_level":3,
  "move_speed_level":1,
  "upgrade_options":[
    {"type":"attack","current_level":2,"next_cost":150,"max_level":10},
    {"type":"attack_speed","current_level":1,"next_cost":120,"max_level":10},
    {"type":"health","current_level":3,"next_cost":180,"max_level":10},
    {"type":"move_speed","current_level":1,"next_cost":110,"max_level":10}
  ]
}
```

升级成功响应为 `{"growth": <同上>, "remaining_coins":850, "cost":150}`。满级属性的 `next_cost` 为 `0`。

## WebSocket

连接成功时，服务端将玩家标为在线并尝试自动恢复活跃对局。客户端应持续发送 `heartbeat`；服务端读取超时为 90 秒。

### 客户端消息

| type | JSON | 行为 |
| --- | --- | --- |
| `heartbeat` | `{"type":"heartbeat"}` | 刷新在线状态和连接活跃时间 |
| `match_start` | `{"type":"match_start","weapon":"sword"}` | 开始匹配；武器为 `sword`、`dagger`、`axe` 或 `bow` |
| `match_cancel` | `{"type":"match_cancel"}` | 取消等待中的匹配 |
| `match_resume` | `{"type":"match_resume"}` | 请求当前玩家的活跃对局 |

### 匹配与恢复消息

等待：

```json
{"type":"match_result","status":"waiting"}
```

匹配或恢复成功：

```json
{
  "type":"match_result",
  "status":"matched",
  "room_name":"room-1001-1002",
  "token":"room-token",
  "battle_node_name":"battle-demo",
  "battle_udp_addr":"127.0.0.1:7001"
}
```

客户端收到 `matched` 后，应使用这些字段发送 UDP `ClientHello`。从战斗主动返回大厅时，应保留该数据并发送 `match_resume`，而不是立即发起新的 `match_start`。

错误和取消：

```json
{"type":"match_error","error":"active match not found"}
```

```json
{"type":"match_canceled"}
```

`active match not found` 表示旧对局已经结束或清理；客户端应清除本地房间数据并允许新匹配。

### 社交与连接事件

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

`connection_replaced` 表示同一玩家在其他 WebSocket 上重新连接，旧连接应停止使用。

## UDP 战斗协议

UDP payload 使用 `proto/battle/v1/session.proto` 的 `ClientPacket` 与 `ServerPacket`。

1. 收到 WebSocket `matched` 后，向 `battle_udp_addr` 发送 `ClientHello(room_name, player_id, token)`。
2. 收到 `ServerHello` 后保存服务器分配的 conversation，开始接收快照。
3. 每 5 秒发送 `ClientHeartbeat`，并按输入状态发送 `ClientInput`。
4. 在 `BATTLE_PHASE_REWARD_SELECTION` 时，使用 snapshot 中的 `current_options` 发送 `ChooseBlessing`。

| 客户端消息 | 必填核心字段 | 说明 |
| --- | --- | --- |
| `ClientHello` | `room_name`、`player_id`、`token` | 首次连接或重连；可重绑 endpoint 与 conversation |
| `ClientInput` | `room_name`、`player_id`、`x`、`y`、`attack_requested`、`dash_requested` | 移动和动作输入 |
| `ClientHeartbeat` | `room_name`、`player_id` | 保持 UDP session 活跃 |
| `ChooseBlessing` | `room_name`、`player_id`、`option_id` | 选择当前候选祝福 |

| 服务端消息 | 说明 |
| --- | --- |
| `ServerHello` | hello 成功和 conversation |
| `GameStart` | 房间开始和完整玩家列表 |
| `WorldSnapshot` | 实体、波次、阶段、经验、祝福和战斗事件 |
| `GameOver` | 房间、完整 roster、结束原因和可选玩家统计 |
| `Error` | token、房间、玩家或输入错误 |

默认超时：有效 UDP 输入或心跳缺失 15 秒时 session 断开；房间内所有 session 都断开 90 秒后，服务器以 `all_players_disconnected` 结束房间。任一玩家重新 hello 会取消该房间的清理计时。

完整字段和枚举以 [session.proto](../proto/battle/v1/session.proto) 为准；控制面和 rcenter RPC 分别见 [battle.proto](../proto/battle/v1/battle.proto) 与 [rcenter.proto](../proto/rcenter/v1/rcenter.proto)。
