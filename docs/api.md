# HTTP API

当前 HTTP API 属于 `logic-server`。HTTP 和 WebSocket 负责客户端入口；账号、玩家、会话和在线状态的数据读写由 `logic-server` 通过 gRPC 交给 `state-server`。匹配请求从 WebSocket 进入 logic-server，再通过 rcenter gRPC 分配 battle-server 房间。

Base URL:

```text
http://localhost:8080
```

`scripts/run.sh` 默认通过 nginx 暴露 `:8080`。如果使用 `START_NGINX=0`，可以直接访问 `http://localhost:8081` 或 `http://localhost:8082`。

认证方式：

```text
Authorization: Bearer <token>
```

token 不放在 JSON body 里。注册和登录成功后服务端返回 token，之后客户端在需要登录态的 HTTP 请求里通过 `Authorization` header 携带。

WebSocket 入口当前使用单独的 header：

```text
token: <token>
```

## GET /health

健康检查。

请求：

```bash
curl -i http://localhost:8080/health
```

成功响应：

```http
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
```

```text
ok server_name = logic-1
```

`server_name` 来自 logic-server 启动参数 `--name`。经过 nginx 访问时，返回值取决于本次请求被转发到哪个 logic-server 实例。

## POST /auth/register

注册账号，同时创建绑定的玩家资料和登录 session。

内部流程：

```text
HTTP register
  -> auth service
  -> state.RegisterAccount gRPC
  -> state-server 创建 account、player、session
```

请求：

```bash
curl -i -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123","nickname":"Alice","avatar":"alice.png","email":"alice@example.com","phone":"13800000000"}'
```

请求 JSON：

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

字段说明：

```text
username: 登录账号名，必填
password: 登录密码，必填
nickname: 玩家展示昵称，必填
avatar:   玩家头像，可为空
email:    邮箱，可为空
phone:    手机号，可为空
```

成功响应：

```http
HTTP/1.1 201 Created
Content-Type: application/json
```

```json
{
  "token": "opaque-session-token",
  "player": {
    "id": 1,
    "nickname": "Alice",
    "avatar": "alice.png",
    "email": "alice@example.com",
    "phone": "13800000000"
  }
}
```

常见错误：

```text
400 Bad Request: 请求 JSON 非法，或必填字段缺失
409 Conflict:    账号已经存在
500 Internal Server Error: 服务内部错误，例如 state-server 或 Redis 异常
```

重复注册响应示例：

```json
{
  "error": "account already exists"
}
```

## POST /auth/login

使用账号和密码登录，成功后创建新的 session token。

请求：

```bash
curl -i -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}'
```

请求 JSON：

```json
{
  "username": "alice",
  "password": "password123"
}
```

成功响应：

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "token": "opaque-session-token",
  "player": {
    "id": 1,
    "nickname": "Alice",
    "avatar": "alice.png",
    "email": "alice@example.com",
    "phone": "13800000000"
  }
}
```

常见错误：

```text
400 Bad Request: 请求 JSON 非法，或必填字段缺失
401 Unauthorized: 账号不存在或密码错误
500 Internal Server Error: 服务内部错误
```

## GET /auth/me

查询当前 token 对应的玩家资料。

请求：

```bash
curl -i http://localhost:8080/auth/me \
  -H 'Authorization: Bearer <token>'
```

成功响应：

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "id": 1,
  "nickname": "Alice",
  "avatar": "alice.png",
  "email": "alice@example.com",
  "phone": "13800000000"
}
```

常见错误：

```text
401 Unauthorized: Authorization header 缺失、格式错误、token 不存在或 session 已失效
500 Internal Server Error: 服务内部错误
```

## POST /auth/logout

登出当前 token。服务端会删除对应 session。

请求：

```bash
curl -i -X POST http://localhost:8080/auth/logout \
  -H 'Authorization: Bearer <token>'
```

成功响应：

```http
HTTP/1.1 204 No Content
```

常见错误：

```text
401 Unauthorized: Authorization header 缺失、格式错误、token 不存在或 session 已失效
500 Internal Server Error: 服务内部错误
```

登出后再次调用 `/auth/me` 应该返回 `401 Unauthorized`。

## GET /ws

建立 WebSocket 连接，并用连接生命周期和 heartbeat 维护玩家在线状态。当前匹配入口也走这个 WebSocket 连接。

握手请求：

```text
GET ws://localhost:8080/ws
token: <token>
```

成功后服务端会：

1. 通过 token 查询 session。
2. WebSocket upgrade 成功后调用 `presence.MarkOnline`。
3. 在 Redis 写入 `game:presence:<player_id>`，包含 `player_id`、`server_name`、`status` 和 `updated_at`，并设置 presence TTL。
4. 在 logic-server 本机 `connManager` 记录当前玩家连接。
5. 持续读取客户端消息，直到客户端断开、读取失败或超过读取超时时间。
6. 收到 heartbeat 后调用 `presence.Refresh`，只在 Redis presence 仍属于当前 `server_name` 时刷新 `updated_at` 和 TTL，同时更新本机连接的 `lastHeartbeatAt`。
7. 连接结束时只有当前连接仍是本机 `connManager` 中的有效连接才调用 `presence.MarkOffline`，避免旧连接断开误清理同一玩家的新连接。

Heartbeat 消息：

```json
{"type":"heartbeat"}
```

客户端建议按固定间隔发送 heartbeat，例如每 30 秒一次。服务端当前 WebSocket read timeout 是 90 秒，超过该时间未收到消息会结束连接并进入离线清理流程。

常见错误：

```text
401 Unauthorized: token header 缺失或 session 无效
101 Switching Protocols: WebSocket 握手成功
```

### heartbeat

客户端发送：

```json
{"type":"heartbeat"}
```

服务端不返回业务消息，只刷新当前玩家的 Redis presence。

### match_start

客户端发送：

```json
{"type":"match_start"}
```

内部流程：

```text
logic WebSocket
  -> match service
  -> rcenter.v1.RCenterService/StartMatch
  -> rcenter waiting queue
  -> battle.v1.BattleControlService/CreateRoom when matched
```

第一个玩家进入队列时，服务端返回：

```json
{
  "type": "match_result",
  "status": "waiting"
}
```

第二个玩家匹配成功时，服务端返回：

```json
{
  "type": "match_result",
  "status": "matched",
  "room_name": "room-...",
  "token": "token-...",
  "battle_node_name": "battle-demo",
  "battle_kcp_addr": "127.0.0.1:7001"
}
```

字段说明：

```text
room_name:        battle-server 中创建的房间名
token:            玩家进入该房间时需要携带的临时 token
battle_node_name: rcenter 选择的 battle 节点名
battle_kcp_addr:  后续客户端连接 battle-server 实时入口的地址
```

如果 rcenter 没有可用 battle node，或 battle-server 创建房间失败，服务端返回：

```json
{
  "type": "match_error",
  "error": "no available BattleNode"
}
```

### match_cancel

客户端发送：

```json
{"type":"match_cancel"}
```

成功响应：

```json
{"type":"match_canceled"}
```

如果玩家不在等待队列，服务端返回：

```json
{
  "type": "match_error",
  "error": "player not waiting"
}
```

其他未知或非法 JSON 消息会被忽略。后续游戏内实时协议可以在这个入口上继续扩展，或拆到 battle-server 的 KCP 入口。

## rcenter gRPC

rcenter gRPC 默认监听：

```text
127.0.0.1:9002
```

proto 文件：

```text
proto/rcenter/v1/rcenter.proto
```

### RegisterBattleNode

注册或刷新一个 battle 节点。rcenter 会在注册时创建并缓存该节点的 battle control gRPC client；同名节点重复注册时会替换旧连接。

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/rcenter/v1/rcenter.proto \
  -d '{"node":{"name":"battle-demo","kcp_addr":"127.0.0.1:7001","control_addr":"127.0.0.1:9101","max_players":100,"active_players":0}}' \
  127.0.0.1:9002 \
  rcenter.v1.RCenterService/RegisterBattleNode
```

### ListBattleNodes

查看当前 rcenter 已注册的 battle 节点：

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/rcenter/v1/rcenter.proto \
  -d '{}' \
  127.0.0.1:9002 \
  rcenter.v1.RCenterService/ListBattleNodes
```

### StartMatch

手动触发匹配。第一次调用通常返回 waiting；第二个不同玩家调用后会触发 battle-server 创建房间。

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/rcenter/v1/rcenter.proto \
  -d '{"player_id":7}' \
  127.0.0.1:9002 \
  rcenter.v1.RCenterService/StartMatch
```

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/rcenter/v1/rcenter.proto \
  -d '{"player_id":8}' \
  127.0.0.1:9002 \
  rcenter.v1.RCenterService/StartMatch
```

匹配成功响应示例：

```json
{
  "result": {
    "status": "matched",
    "roomName": "room-...",
    "token": "token-...",
    "battleNodeName": "battle-demo",
    "battleKcpAddr": "127.0.0.1:7001",
    "playerIds": ["7", "8"]
  }
}
```

## battle control gRPC

C++ battle-server control gRPC 默认监听：

```text
127.0.0.1:9101
```

proto 文件：

```text
proto/battle/v1/battle.proto
```

### CreateRoom

rcenter 匹配成功后会调用这个 RPC。也可以手动验证：

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/battle/v1/battle.proto \
  -d '{"room_name":"room-e2e","token":"token-e2e","player_ids":[7,8]}' \
  127.0.0.1:9101 \
  battle.v1.BattleControlService/CreateRoom
```

成功响应：

```json
{
  "status": "CREATE_ROOM_STATUS_OK",
  "message": "room created"
}
```

重复创建同名房间会返回：

```json
{
  "status": "CREATE_ROOM_STATUS_ALREADY_EXISTS",
  "message": "room already exists"
}
```

### JoinRoom

当前 JoinRoom 还是 control gRPC 验证入口。后续接入 KCP 后，客户端首个 KCP 业务包会携带 `room_name`、`player_id` 和 `token`，再复用同一套房间校验逻辑。

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/battle/v1/battle.proto \
  -d '{"room_name":"room-e2e","token":"token-e2e","player_id":7}' \
  127.0.0.1:9101 \
  battle.v1.BattleControlService/JoinRoom
```

成功响应：

```json
{
  "status": "JOIN_ROOM_STATUS_OK",
  "message": "player joined"
}
```

同一玩家重复 join 会返回：

```json
{
  "status": "JOIN_ROOM_STATUS_ALREADY_JOINED",
  "message": "player already joined"
}
```

## Full Manual Check

建议手动验证顺序：

1. 启动 Redis。
2. 执行 `bash scripts/run.sh`。
3. 调用 `/health`，确认 `logic-server` 可访问。
4. 调用 `/auth/register`，确认返回 `201 Created` 和 token。
5. 使用相同 username 再注册一次，确认返回 `409 Conflict`。
6. 调用 `/auth/login`，确认返回新的 token。
7. 用 token 调用 `/auth/me`，确认返回玩家资料。
8. 用 token 连接 `/ws`，确认 Redis 出现 `game:presence:<player_id>`。
9. 发送 `{"type":"heartbeat"}`，确认 Redis 中 presence 的 `updated_at` 和 TTL 被刷新。
10. 启动 battle-server，并用 `RegisterBattleNode` 注册到 rcenter。
11. 两个不同玩家分别发送 `{"type":"match_start"}`，确认第二次返回 `matched`，并且 battle-server 创建房间成功。
12. 断开 `/ws`，确认对应 presence 被清理。
13. 调用 `/auth/logout`，确认返回 `204 No Content`。
14. 再用同一个 token 调用 `/auth/me`，确认返回 `401 Unauthorized`。

可以用下面命令查看 Redis 当前写入的数据：

```bash
redis-cli keys 'game:*'
```

可以用下面命令清理本项目数据：

```bash
bash scripts/reset_redis.sh
```
