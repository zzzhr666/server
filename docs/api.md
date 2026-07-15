# HTTP API 文档

本文档描述当前 `game-server` 已实现的 HTTP 接口。当前版本主要服务游戏外逻辑，协议为 HTTP + JSON，默认地址为:

```text
http://localhost:8080
```

当前后端实现采用模块化单体结构:

```text
HTTP API
  -> player.Service / room.Service
  -> player.Repository / room.Repository
  -> Redis
```

接口路径保持稳定，后续可以把内部 repository 替换为 RPC/gRPC client，而不改变客户端请求格式。

## 通用约定

### 请求格式

除健康检查外，请求体使用 JSON:

```http
Content-Type: application/json
```

### 响应格式

成功响应通常返回 JSON。无响应体的成功操作返回 `204 No Content`。

错误响应统一为:

```json
{
  "error": "error message"
}
```

### 常见状态码

| 状态码 | 含义 |
| --- | --- |
| 200 | 请求成功 |
| 201 | 创建成功 |
| 204 | 操作成功，无响应体 |
| 400 | 请求参数或 JSON 格式错误 |
| 404 | 玩家或房间不存在 |
| 409 | 当前状态冲突，例如重复加入房间 |
| 500 | 服务内部错误 |

### 数据持久化

当前接口数据写入 Redis:

| 数据 | Redis 结构 |
| --- | --- |
| 玩家 ID 自增 | `game:next_player_id` |
| 玩家信息 | `game:player:{id}` hash |
| 房间 ID 自增 | `game:next_room_id` |
| 房间信息 | `game:room:{id}` hash |
| 房间成员 | `game:room:{id}:players` set |
| 房间列表 | `game:rooms` set |

如果手动验证接口，建议在本地开发 Redis 中清理 `game:*` 测试数据，避免自增 ID 影响示例结果。

## 健康检查

### GET /health

用于服务探活，后续可供 Nginx 或部署平台做健康检查。

#### 响应示例

```text
ok
```

#### 状态码

| 状态码 | 说明 |
| --- | --- |
| 200 | 服务存活 |

## 玩家接口

### POST /players

创建玩家。

#### 请求体

```json
{
  "name": "alice"
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 玩家名称，不能为空 |

#### 成功响应

状态码: `201 Created`

```json
{
  "id": 1,
  "name": "alice"
}
```

#### 错误响应

| 状态码 | 场景 |
| --- | --- |
| 400 | JSON 格式错误或玩家名为空 |
| 500 | 服务内部错误 |

#### curl 示例

```bash
curl -X POST http://localhost:8080/players \
  -H 'Content-Type: application/json' \
  -d '{"name":"alice"}'
```

### GET /players/{id}

查询玩家信息。

#### 路径参数

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| id | int64 | 玩家 ID |

#### 成功响应

状态码: `200 OK`

```json
{
  "id": 1,
  "name": "alice"
}
```

#### 错误响应

| 状态码 | 场景 |
| --- | --- |
| 400 | 玩家 ID 不是合法数字 |
| 404 | 玩家不存在 |
| 500 | 服务内部错误 |

#### curl 示例

```bash
curl http://localhost:8080/players/1
```

## 房间接口

### POST /rooms

创建房间。创建成功后，房主会自动加入该房间。

#### 请求体

```json
{
  "owner_id": 1
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| owner_id | int64 | 是 | 房主玩家 ID |

#### 成功响应

状态码: `201 Created`

```json
{
  "id": 1,
  "owner_id": 1,
  "players": [1]
}
```

#### 错误响应

| 状态码 | 场景 |
| --- | --- |
| 400 | JSON 格式错误 |
| 404 | 房主玩家不存在 |
| 500 | 服务内部错误 |

#### curl 示例

```bash
curl -X POST http://localhost:8080/rooms \
  -H 'Content-Type: application/json' \
  -d '{"owner_id":1}'
```

### GET /rooms/{id}

查询单个房间。

#### 路径参数

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| id | int64 | 房间 ID |

#### 成功响应

状态码: `200 OK`

```json
{
  "id": 1,
  "owner_id": 1,
  "players": [1, 2]
}
```

#### 错误响应

| 状态码 | 场景 |
| --- | --- |
| 400 | 房间 ID 不是合法数字 |
| 404 | 房间不存在 |
| 500 | 服务内部错误 |

#### curl 示例

```bash
curl http://localhost:8080/rooms/1
```

### GET /rooms

查询房间列表。

#### 成功响应

状态码: `200 OK`

```json
{
  "rooms": [
    {
      "id": 1,
      "owner_id": 1,
      "players": [1, 2]
    }
  ]
}
```

#### curl 示例

```bash
curl http://localhost:8080/rooms
```

### POST /rooms/{id}/join

玩家加入房间。

#### 路径参数

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| id | int64 | 房间 ID |

#### 请求体

```json
{
  "player_id": 2
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| player_id | int64 | 是 | 加入房间的玩家 ID |

#### 成功响应

状态码: `204 No Content`

#### 错误响应

| 状态码 | 场景 |
| --- | --- |
| 400 | 房间 ID 非法或 JSON 格式错误 |
| 404 | 玩家或房间不存在 |
| 409 | 玩家已经在房间中 |
| 500 | 服务内部错误 |

#### curl 示例

```bash
curl -X POST http://localhost:8080/rooms/1/join \
  -H 'Content-Type: application/json' \
  -d '{"player_id":2}'
```

### POST /rooms/{id}/leave

玩家离开房间。

#### 路径参数

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| id | int64 | 房间 ID |

#### 请求体

```json
{
  "player_id": 2
}
```

#### 成功响应

状态码: `204 No Content`

#### 错误响应

| 状态码 | 场景 |
| --- | --- |
| 400 | 房间 ID 非法或 JSON 格式错误 |
| 404 | 玩家或房间不存在 |
| 409 | 玩家不在房间中 |
| 500 | 服务内部错误 |

#### curl 示例

```bash
curl -X POST http://localhost:8080/rooms/1/leave \
  -H 'Content-Type: application/json' \
  -d '{"player_id":2}'
```

## 完整调用流程示例

```bash
# 1. 创建房主
curl -X POST http://localhost:8080/players \
  -H 'Content-Type: application/json' \
  -d '{"name":"alice"}'

# 2. 创建另一个玩家
curl -X POST http://localhost:8080/players \
  -H 'Content-Type: application/json' \
  -d '{"name":"bob"}'

# 3. alice 创建房间
curl -X POST http://localhost:8080/rooms \
  -H 'Content-Type: application/json' \
  -d '{"owner_id":1}'

# 4. bob 加入房间
curl -X POST http://localhost:8080/rooms/1/join \
  -H 'Content-Type: application/json' \
  -d '{"player_id":2}'

# 5. 查询房间
curl http://localhost:8080/rooms/1
```

## 后续接口规划

### 登录模块

- `POST /auth/login`: 游客登录或账号登录
- `POST /auth/logout`: 登出
- `GET /me`: 查询当前登录玩家

### 好友模块

- `POST /friends/requests`: 发送好友申请
- `GET /friends/requests`: 查看好友申请
- `POST /friends/requests/{id}/accept`: 接受好友申请
- `DELETE /friends/{id}`: 删除好友
- `GET /friends`: 好友列表

### 房间增强

- `PATCH /rooms/{id}`: 修改房间配置
- `POST /rooms/{id}/kick`: 踢出玩家
- `POST /rooms/{id}/transfer-owner`: 转让房主
- `POST /rooms/{id}/ready`: 玩家准备
- `POST /rooms/{id}/start`: 开始游戏

计划新增字段:

| 字段 | 说明 |
| --- | --- |
| status | 房间状态，例如 `waiting`、`playing`、`closed` |
| max_players | 房间最大人数 |
| mode | 游戏模式 |
| created_at | 创建时间 |

### 匹配模块

- `POST /match/queue`: 加入匹配队列
- `DELETE /match/queue`: 取消匹配
- `GET /match/status`: 查询匹配状态

## 版本演进建议

为了后续接入 Nginx、RPC 和 gRPC/Protobuf，建议保持以下约束:

- HTTP API 对外保持稳定，内部实现可以从 Redis repository 替换为 RPC/gRPC client。
- 请求和响应结构尽量接近未来的 Protobuf message，减少迁移成本。
- 错误响应后续可以升级为 `{ "code": "...", "message": "..." }`，便于前端和客户端处理。
- 每个业务模块优先沉淀清晰的 service/repository 接口，再决定是否拆成独立服务。
