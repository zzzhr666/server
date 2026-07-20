# HTTP API

当前 HTTP API 属于 `logic-server`。HTTP 和 WebSocket 负责客户端入口；账号、玩家、会话和在线状态的数据读写由 `logic-server` 通过 gRPC 交给 `state-server`。

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

建立 WebSocket 连接，并用连接生命周期和 heartbeat 维护玩家在线状态。

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

当前 `/ws` 只定义了 heartbeat 业务消息；其他未知或非法 JSON 消息会被忽略。后续游戏内实时协议可以在这个入口上继续扩展，或拆到 room-server。

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
10. 断开 `/ws`，确认对应 presence 被清理。
11. 调用 `/auth/logout`，确认返回 `204 No Content`。
12. 再用同一个 token 调用 `/auth/me`，确认返回 `401 Unauthorized`。

可以用下面命令查看 Redis 当前写入的数据：

```bash
redis-cli keys 'game:*'
```

可以用下面命令清理本项目数据：

```bash
bash scripts/reset_redis.sh
```
