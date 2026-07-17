# HTTP API

当前 HTTP API 属于 `logic-server`。HTTP 只负责登录相关流程和基础健康检查；数据状态读写由 `logic-server` 通过 RPC 交给 `state-server`。

Base URL:

```text
http://localhost:8080
```

认证方式：

```text
Authorization: Bearer <token>
```

token 不放在 JSON body 里。注册和登录成功后服务端返回 token，之后客户端在需要登录态的请求里通过 `Authorization` header 携带。

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
ok
```

## POST /auth/register

注册账号，同时创建绑定的玩家资料和登录 session。

内部流程：

```text
HTTP register
  -> auth service
  -> state.RegisterAccount RPC
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

## Full Manual Check

建议手动验证顺序：

1. 启动 Redis。
2. 执行 `bash scripts/run.sh`。
3. 调用 `/health`，确认 `logic-server` 可访问。
4. 调用 `/auth/register`，确认返回 `201 Created` 和 token。
5. 使用相同 username 再注册一次，确认返回 `409 Conflict`。
6. 调用 `/auth/login`，确认返回新的 token。
7. 用 token 调用 `/auth/me`，确认返回玩家资料。
8. 调用 `/auth/logout`，确认返回 `204 No Content`。
9. 再用同一个 token 调用 `/auth/me`，确认返回 `401 Unauthorized`。

可以用下面命令查看 Redis 当前写入的数据：

```bash
redis-cli keys 'game:*'
```

可以用下面命令清理本项目数据：

```bash
bash scripts/reset_redis.sh
```
