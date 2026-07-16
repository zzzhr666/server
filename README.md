# Game Server Demo

Go 编写的游戏服务器 demo，当前版本聚焦游戏外逻辑: 玩家资料、房间、加入/离开房间、准备状态、开始游戏、健康检查，以及已完成服务层但尚未接入 HTTP 的账号认证能力。服务使用 HTTP + JSON 对外提供接口，使用 Redis 存储玩家、房间和 auth 数据。

## 架构

```text
Client
  |
  v
HTTP JSON API
  |
  v
internal/httpapi
  |
  v
player.Service / room.Service
  |
  v
player.Repository / room.Repository
  |
  v
Redis
```

`auth.Service` 当前已完成注册、登录、登出和 session 查询的业务层与 Redis repository，HTTP 路由会在下一阶段接入。

## 项目结构

```text
.
├── cmd/game-server/          # 服务启动入口
├── internal/config/          # 项目默认配置
├── internal/httpapi/         # HTTP 路由、请求解析、响应编码
├── internal/auth/            # 账号、密码哈希、登录会话、Redis repository
├── internal/player/          # 玩家模型、服务接口、Redis repository
├── internal/room/            # 房间模型、服务接口、Redis repository
├── internal/redisdb/         # Redis 客户端和事务重试封装
├── docs/api.md               # HTTP 接口文档
├── docs/architecture.md      # 架构和模块扩展说明
├── go.mod
└── README.md
```

## Quickstart

启动 Redis:

```bash
redis-server
```

或使用 Docker:

```bash
docker run --rm -p 6379:6379 redis:latest
```

启动服务:

```bash
go run ./cmd/game-server
```

服务默认监听:

```text
:8080
```

Redis 默认连接:

```text
127.0.0.1:6379
```

健康检查:

```bash
curl http://localhost:8080/health
```

清理本项目 Redis 数据:

```bash
bash scripts/reset_redis.sh
```

创建玩家:

```bash
curl -X POST http://localhost:8080/players \
  -H 'Content-Type: application/json' \
  -d '{"name":"alice","avatar":"alice.png"}'
```

创建房间:

```bash
curl -X POST http://localhost:8080/rooms \
  -H 'Content-Type: application/json' \
  -d '{"owner_id":1}'
```

查询房间详情:

```bash
curl http://localhost:8080/rooms/1
```

## 测试

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

## 文档

- [HTTP API](docs/api.md)
- [Architecture](docs/architecture.md): 当前模块边界、Redis 结构和后续多服务升级路线
