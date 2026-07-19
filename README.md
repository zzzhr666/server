# Game Server Demo

这是一个 Go 游戏服务器 demo。当前阶段的重点不是完整玩法，而是把服务拆成多个进程，并跑通“客户端访问 logic-server，logic-server 通过 gRPC 调用 state-server，state-server 统一操作 Redis”的基础链路。

当前已经可用的链路是：

```text
Client
  -> nginx reverse proxy (:8080, optional)
  -> logic-server HTTP/WebSocket (:8081 / :8082)
  -> internal/logic/*
  -> internal/state/grpcclient
  -> state-server gRPC (:9001)
  -> internal/state/service
  -> internal/state/redisstore
  -> Redis
```

## 当前进程

```text
cmd/
├── logic-server/   # HTTP/WebSocket 入口，当前负责认证、玩家查询和在线状态连接
├── state-server/   # 状态服务进程，通过 protobuf/gRPC 暴露 Redis 数据操作
├── rcenter-server/ # 资源调度中心骨架，后续负责匹配、房间服分配、负载管理
└── room-server/    # 游戏服务骨架，后续承载游戏内会话和实时逻辑
```

### logic-server

`logic-server` 是现在对客户端暴露 HTTP API 和 WebSocket 连接的进程。

当前接口：

- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`
- `GET /ws`

它不直接操作 Redis，而是通过 `internal/state/grpcclient` 调用 `state-server`。启动时可以用 `-p` 或 `--port` 指定 HTTP 端口，用 `--name` 指定实例名；实例名会写入在线状态，方便判断玩家当前挂在哪个 logic-server。

### state-server

`state-server` 是当前唯一直接操作 Redis 的业务进程。

启动后它会：

1. 连接 Redis。
2. 创建 `internal/state/redisstore.Store`。
3. 创建 `internal/state/service.Service`。
4. 把 state service 注册到 gRPC `StateService`。
5. 监听 `127.0.0.1:9001`。

### nginx

`scripts/run.sh` 默认会启动一个本地 nginx 反向代理，监听 `:8080`，并把请求转发到两个 logic-server 实例：

```text
127.0.0.1:8081 -> logic-1
127.0.0.1:8082 -> logic-2
```

nginx 配置位于 `deploy/nginx/logic.conf`，已经包含 WebSocket upgrade 相关 header。

### rcenter-server 和 room-server

这两个进程目前只是骨架。

- `rcenter-server` 后续作为资源调度中心，负责匹配、分配 room-server、记录房间服负载。
- `room-server` 后续作为游戏服务进程，负责承载游戏内会话和实时消息。

## 代码结构

```text
internal/
├── contract/
│   ├── state/      # state-server 对外暴露的业务模型、错误和 Client 接口
│   └── statepb/    # proto/state/v1/state.proto 生成的 gRPC 代码
├── logic/
│   ├── auth/       # 注册、登录、登出、会话校验等认证业务
│   ├── player/     # 玩家资料业务
│   ├── presence/   # 玩家在线状态业务
│   └── httpapi/    # logic-server 的 HTTP/WebSocket 路由和 handler
├── platform/
│   ├── config/     # 进程默认配置
│   └── redisdb/    # Redis client 创建
└── state/
    ├── grpcclient/ # logic-server 使用的 state gRPC 客户端适配层
    ├── grpcserver/ # state-server 使用的 gRPC 服务适配层
    ├── redisstore/ # Redis 存储实现
    ├── service/    # state 业务编排
    └── stateproto/ # state contract 与 protobuf message 的转换
```

几个重要边界：

- `internal/logic/*` 写认证、玩家、在线状态等“业务逻辑”，不写 Redis 命令。
- `internal/state/service` 写 state 服务的业务编排，不处理 HTTP 或 WebSocket。
- `internal/state/redisstore` 只负责 Redis 读写和 Redis 侧的一致性控制。
- `internal/contract/state` 是 logic-server 和 state-server 之间共享的业务契约。
- `proto/state/v1/state.proto` 是 gRPC 传输契约；生成代码放在 `internal/contract/statepb`。

## 配置

默认配置在 `internal/platform/config`：

```text
HTTPAddr:      :8080
StateGRPCAddr: 127.0.0.1:9001
Redis:         127.0.0.1:6379
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

`scripts/run.sh` 会先启动 `state-server`，等待 `127.0.0.1:9001` 可连接后，再启动两个 `logic-server` 实例。默认还会用 `sudo nginx` 启动本地 nginx 反向代理到 `:8080`。

如果不想启动 nginx，可以只启动 state-server 和两个 logic-server：

```bash
START_NGINX=0 bash scripts/run.sh
```

也可以手动分多个终端启动：

```bash
redis-server
```

```bash
go run ./cmd/state-server
```

```bash
go run ./cmd/logic-server -p 8081 --name logic-1
```

```bash
go run ./cmd/logic-server -p 8082 --name logic-2
```

健康检查：

```bash
curl http://localhost:8080/health
```

如果没有启动 nginx，直接访问某个 logic-server：

```bash
curl http://localhost:8081/health
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

WebSocket 在线状态：

```text
GET ws://localhost:8080/ws
Header: token: <token>
```

连接建立后，服务端会把玩家标记为在线；连接断开后，如果 Redis 中记录的 `server_name` 仍然等于当前 logic-server 实例名，就清理该玩家在线状态。

查看 Redis 数据：

```bash
redis-cli keys 'game:*'
```

清理本项目 Redis 数据：

```bash
bash scripts/reset_redis.sh
```

## Proto 生成

修改 `proto/state/v1/state.proto` 后重新生成 gRPC 代码：

```bash
bash scripts/generate_proto.sh
```

生成结果会写入 `internal/contract/statepb`。

## 测试

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

## 当前限制

- `logic-server` 和 `state-server` 之间已经迁移到 protobuf/gRPC，但还没有接入 TLS、服务发现或连接池策略。
- `scripts/run.sh` 默认依赖本机 nginx 和 sudo 权限；没有 nginx 时可以用 `START_NGINX=0` 跑两个 logic-server 实例。
- `state-server` 已经独立成进程，Redis 写入使用事务 pipeline 和乐观锁控制核心冲突，但还不是商业级分布式事务方案。
- 在线状态目前只记录 `online`、`server_name` 和更新时间，还没有心跳续期接口、好友可见性或状态广播。
- `rcenter-server` 和 `room-server` 还只是骨架。
