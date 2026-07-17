# Game Server Demo

这是一个 Go 游戏服务器 demo。当前阶段的重点不是完整玩法，而是把服务器拆成多个进程，并先跑通“登录 HTTP 服务通过 RPC 调用状态服务，状态服务统一操作 Redis”的基础链路。

当前已经可用的链路是：

```text
Client
  -> logic-server HTTP
  -> internal/logic/*
  -> state RPC client
  -> state-server net/rpc
  -> internal/state/service
  -> internal/state/redisstore
  -> Redis
```

## 当前进程

```text
cmd/
├── logic-server/   # HTTP 入口，当前负责注册、登录、登出、查询当前玩家
├── state-server/   # 状态服务进程，当前负责通过 RPC 暴露 Redis 数据操作
├── rcenter-server/ # 资源调度中心骨架，后续负责匹配、房间服分配、负载管理
└── room-server/    # 游戏服务骨架，后续承载游戏内会话和实时逻辑
```

### logic-server

`logic-server` 是现在唯一对客户端暴露 HTTP API 的进程。

当前接口：

- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`

它不直接操作 Redis，而是通过 `internal/state/rpcclient` 调用 `state-server`。

### state-server

`state-server` 是当前唯一直接操作 Redis 的业务进程。

启动后它会：

1. 连接 Redis。
2. 创建 `internal/state/redisstore.Store`。
3. 创建 `internal/state/service.Service`。
4. 把 state service 注册成 `net/rpc` 服务 `StateService`。
5. 监听 `127.0.0.1:9001`。

### rcenter-server 和 room-server

这两个进程目前只是骨架。

- `rcenter-server` 后续作为资源调度中心，负责匹配、分配 room-server、记录房间服负载。
- `room-server` 后续作为游戏服务进程，负责承载游戏内会话和实时消息。

## 代码结构

```text
internal/
├── contract/
│   ├── rpc/        # RPC 契约说明占位
│   └── state/      # state-server 对外暴露的数据模型、错误和 Client 接口
├── logic/
│   ├── auth/       # 注册、登录、登出、会话校验等登录业务
│   ├── player/     # 玩家资料业务
│   └── httpapi/    # logic-server 的 HTTP 路由、请求响应协议和 handler
├── platform/
│   ├── config/     # 进程默认配置
│   └── redisdb/    # Redis client 创建
└── state/
    ├── redisstore/ # Redis 存储实现
    ├── rpcclient/  # logic-server 使用的 state RPC 客户端
    ├── rpcserver/  # state-server 使用的 RPC 服务适配层
    └── service/    # state 业务编排、锁和组合操作
```

几个重要边界：

- `internal/logic/*` 写登录、玩家等“业务逻辑”，不写 Redis 命令。
- `internal/state/service` 写跨数据的组合操作和锁，例如注册时创建账号、玩家、会话。
- `internal/state/redisstore` 只负责 Redis 读写，不负责业务流程。
- `internal/contract/state` 是 logic-server 和 state-server 之间共享的状态服务契约。

## 配置

默认配置在 `internal/platform/config`：

```text
HTTPAddr:     :8080
StateRPCAddr: 127.0.0.1:9001
Redis:        127.0.0.1:6379
```

当前 demo 先使用代码默认值，后续可以再扩展成环境变量或配置文件。

## 启动

先启动 Redis：

```bash
redis-server
```

然后启动项目：

```bash
bash scripts/run.sh
```

`scripts/run.sh` 会先启动 `state-server`，等待 `127.0.0.1:9001` 可连接后，再启动 `logic-server`。

也可以手动分三个终端启动：

```bash
redis-server
```

```bash
go run ./cmd/state-server
```

```bash
go run ./cmd/logic-server
```

健康检查：

```bash
curl http://localhost:8080/health
```

## 手动验证

注册：

```bash
curl -i -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123","nickname":"Alice"}'
```

登录：

```bash
curl -i -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}'
```

查询当前玩家：

```bash
curl -i http://localhost:8080/auth/me \
  -H 'Authorization: Bearer <token>'
```

登出：

```bash
curl -i -X POST http://localhost:8080/auth/logout \
  -H 'Authorization: Bearer <token>'
```

查看 Redis 数据：

```bash
redis-cli keys 'game:*'
```

清理本项目 Redis 数据：

```bash
bash scripts/reset_redis.sh
```

## 测试

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

## 当前限制

- 当前 RPC 使用 Go 标准库 `net/rpc`，还没有接入 protobuf/gRPC。
- `state-server` 已经独立成进程，但注册流程失败后的回滚还不是商业级实现。
- `rcenter-server` 和 `room-server` 还只是骨架。
- 好友、在线状态、匹配、自定义 lobby、游戏内 world/session 等模块还没有实现。

后续如果迁移到 gRPC，优先替换 `internal/state/rpcclient` 和 `internal/state/rpcserver`，并新增 `.proto` 契约；`logic` 业务层、`state/service` 和 `state/redisstore` 应尽量保持稳定。
