# Architecture

项目正在从单进程 HTTP + Redis demo 迁移成多进程游戏服务器 demo。当前已经完成的核心变化是：`logic-server` 不再直接访问 Redis，而是通过 protobuf/gRPC 调用独立的 `state-server`，由 `state-server` 统一操作 Redis。

旧的 Go `net/rpc` client/server 适配层已经删除，仓库里只保留当前 gRPC 通信链路。

## Runtime View

当前可运行链路：

```text
Client
  |
  | HTTP JSON / WebSocket
  v
nginx (:8080, optional)
  |
  v
logic-server (:8081, logic-1) / logic-server (:8082, logic-2)
  |
  | auth / player / presence service
  v
logic state repository adapter
  |
  | statecontract.Client / statecontract.PresenceClient
  v
state grpcclient
  |
  | protobuf/gRPC, state.v1.StateService
  v
state-server (127.0.0.1:9001)
  |
  | grpcserver adapter
  v
state service
  |
  | accountStore / playerStore / sessionStore / presenceStore
  v
redisstore
  |
  v
Redis (127.0.0.1:6379)
```

这条链路的意义是把“客户端入口”和“数据状态操作”拆开：

- `logic-server` 负责 HTTP、WebSocket、认证业务、玩家资料业务和在线状态业务。
- `state-server` 负责状态数据读写和跨数据组合操作。
- Redis 只被 `state-server` 直接访问。
- nginx 只负责本地 demo 的入口代理和 WebSocket upgrade 转发。

## Process Responsibilities

### logic-server

入口：`cmd/logic-server/main.go`

职责：

- 启动 HTTP/WebSocket 服务。
- 注册 `/health`、`/auth/*` 和 `/ws` 路由。
- 创建 `auth.Service`、`player.Service` 和 `presence.Service`。
- 通过 gRPC 连接 `state-server`。
- 使用 `internal/state/grpcclient.Client` 作为 state client。
- 用 `--name` 标识当前实例，写入 presence 的 `server_name`。

它依赖 state 契约，但不关心 state 的真实存储是 Redis、MySQL，还是别的服务。

### state-server

入口：`cmd/state-server/main.go`

职责：

- 连接 Redis。
- 创建 Redis store。
- 创建 state service。
- 把 state service 注册成 gRPC `StateService`。
- 监听 `127.0.0.1:9001`。

所有跨账号、玩家、会话的组合写操作，都应该尽量放在 `state-server` 内部做成一个粗粒度方法，而不是让 `logic-server` 连续调用多个细粒度 gRPC 方法。

例如注册账号现在使用：

```text
logic auth service
  -> state.RegisterAccount(...)
  -> state-server 内部创建 player、account、session
```

这样比下面这种方式更容易控制并发和一致性：

```text
logic-server
  -> NextPlayerID
  -> CreatePlayer
  -> CreateAccount
  -> CreateSession
```

### nginx

配置：`deploy/nginx/logic.conf`

职责：

- 监听 `:8080`。
- 转发 HTTP 请求到 `127.0.0.1:8081` 和 `127.0.0.1:8082`。
- 保留 WebSocket upgrade 相关 header。

nginx 只属于当前本地 demo 的启动体验，不进入 Go 业务边界。

### rcenter-server

入口：`cmd/rcenter-server/main.go`

当前状态：骨架。

后续职责方向：

- 管理 room-server 注册和负载。
- 承担匹配服务或资源调度中心职责。
- 为自定义 lobby 或匹配请求分配合适的 room-server。
- 生成或校验进入游戏服务所需的调度信息。

### room-server

入口：`cmd/room-server/main.go`

当前状态：骨架。

后续职责方向：

- 承载游戏内会话。
- 处理实时消息。
- 和 `state-server` 交互读取玩家状态、写入结算结果。
- 和 `rcenter-server` 交互汇报负载、创建或销毁游戏会话。

## Package Layout

```text
cmd/
├── logic-server/
├── state-server/
├── rcenter-server/
└── room-server/

proto/
└── state/v1/state.proto

internal/
├── contract/
│   ├── state/
│   └── statepb/
├── logic/
│   ├── auth/
│   ├── player/
│   ├── presence/
│   └── httpapi/
├── platform/
│   ├── config/
│   └── redisdb/
└── state/
    ├── grpcclient/
    ├── grpcserver/
    ├── redisstore/
    ├── service/
    └── stateproto/
```

### internal/contract/state

这是 state-server 对外暴露的共享业务契约。

主要内容：

- `Account`
- `Player`
- `Session`
- `Presence`
- `RegisterAccountInput`
- `RegisterAccountResult`
- `Client` 接口
- `PresenceClient` 接口
- state 级错误，例如 `ErrAccountExists`、`ErrSessionNotFound`、`ErrPresenceNotFound`

`logic-server` 依赖这个接口，不依赖 state-server 的具体实现。

### internal/contract/statepb

这是 `proto/state/v1/state.proto` 生成的 protobuf/gRPC 代码。

主要内容：

- protobuf message。
- `StateServiceClient`。
- `StateServiceServer`。
- gRPC 方法描述。

业务代码不应该直接把 protobuf message 泄漏到 logic 层；protobuf 和业务模型之间的转换放在 `internal/state/stateproto`。

### internal/logic/auth

认证业务层。

主要职责：

- 校验注册和登录输入。
- 生成 bcrypt 密码哈希。
- 校验密码。
- 生成 session token。
- 调用 state repository 创建账号、会话，或读取账号、会话。
- 把 state 错误转换成 auth 业务错误。

`state_repository.go` 是适配层：它把 auth service 需要的仓储操作转成 `statecontract.Client` 调用。

### internal/logic/player

玩家资料业务层。

当前主要负责：

- 根据玩家 ID 查询玩家资料。
- 把 state player 模型转换成 logic player 模型。

`state_repository.go` 是适配层：它把 player service 需要的仓储操作转成 `statecontract.Client` 调用。

### internal/logic/presence

在线状态业务层。

当前主要负责：

- WebSocket 建连后标记玩家在线。
- 记录玩家所在 logic-server 实例名。
- WebSocket 断开后清理仍属于当前实例的在线状态。
- 把 state presence 错误转换成 logic presence 错误。

presence 的默认 TTL 是 2 分钟。当前实现会在连接建立时写入 TTL，但还没有心跳续期；长连接在线状态的续期策略后续需要补齐。

### internal/logic/httpapi

HTTP/WebSocket 适配层。

主要职责：

- 定义 HTTP 和 WebSocket 路由。
- 解析 JSON 请求。
- 读取 `Authorization: Bearer <token>`。
- 读取 WebSocket 握手使用的 `token` header。
- 调用 logic service。
- 把业务错误映射为 HTTP 状态码。
- 输出 JSON 响应。
- 在 WebSocket 生命周期里调用 presence service。

HTTP 层不直接访问 Redis，也不直接调用 generated gRPC client。

### internal/state/grpcserver

state-server 使用的 gRPC 适配层。

主要职责：

- 实现 generated `statepb.StateServiceServer`。
- 把 protobuf request 转换成 `statecontract` 模型。
- 调用 `statecontract.Client` 和 `statecontract.PresenceClient`。
- 把 state sentinel error 映射成 gRPC status code。

### internal/state/grpcclient

logic-server 使用的 gRPC 客户端适配层。

主要职责：

- 持有 generated `statepb.StateServiceClient`。
- 实现 `statecontract.Client` 和 `statecontract.PresenceClient`。
- 把 `statecontract` 模型转换成 protobuf request。
- 把 gRPC status error 映射回 state sentinel error。

错误映射很重要，因为 logic 层依赖 `errors.Is` 判断 `statecontract.ErrAccountExists`、`statecontract.ErrSessionNotFound` 等哨兵错误。

### internal/state/stateproto

protobuf message 与 state contract 模型的转换层。

这个包把 `timestamppb.Timestamp`、`durationpb.Duration`、`statepb.Account`、`statepb.Player`、`statepb.Session`、`statepb.Presence` 转成业务模型，避免 protobuf 类型向业务层扩散。

### internal/state/service

state 业务层。

主要职责：

- 暴露账号、玩家、会话和在线状态操作。
- 组合 store 接口，隔离上层与具体存储实现。
- 让 gRPC server 面向统一的 state client。

当前跨资源注册操作已经下沉到 `registrationStore.RegisterAccount`，由 Redis store 用 Redis 事务能力处理冲突。

### internal/state/redisstore

Redis 存储实现。

主要职责：

- 把 state 模型存入 Redis。
- 从 Redis 读取 state 模型。
- 维护 player ID 自增键。
- 用 Redis `WATCH` 和 transaction pipeline 处理账号创建、注册和 presence 清理的冲突。

当前 key 大致包括：

```text
game:account:<username>
game:player:<id>
game:session:<token>
game:presence:<player_id>
game:next_player_id
```

`RegisterAccount` 会检查账号是否存在，生成玩家 ID，写入玩家、账号和 session。玩家 ID 使用 Redis 自增，因此在并发冲突或失败重试时允许出现 ID 空洞。

`ClearPresence` 会先比较 Redis 中保存的 `server_name`，只有它仍然等于当前 logic-server 实例名时才删除 key。这样可以避免旧连接断开时误删同一玩家在新连接上写入的在线状态。

如果以后从 Redis 换成 MySQL，优先新增一个 MySQL store，让它实现 `state/service` 需要的 store 接口。理论上 `logic` 层不应该被影响。

## Dependency Direction

依赖方向应该保持为：

```text
cmd/logic-server
  -> internal/logic/*
  -> internal/contract/state
  -> internal/state/grpcclient
  -> internal/contract/statepb

cmd/state-server
  -> internal/state/grpcserver
  -> internal/state/service
  -> internal/state/redisstore
  -> internal/platform/redisdb
```

不建议出现：

```text
internal/logic/* -> internal/state/redisstore
internal/logic/* -> internal/platform/redisdb
internal/logic/httpapi -> internal/state/grpcclient
internal/logic/* -> internal/contract/statepb
```

原因是 logic 的业务代码应该通过接口表达“我要什么状态能力”，而不是知道 Redis、protobuf 或 gRPC 的细节。

## Error Flow

以重复注册为例：

```text
redisstore
  -> statecontract.ErrAccountExists
state service
  -> statecontract.ErrAccountExists
grpcserver
  -> codes.AlreadyExists
grpcclient
  -> map back to statecontract.ErrAccountExists
logic auth state repository
  -> auth.ErrAccountExists
httpapi
  -> HTTP 409 {"error":"account already exists"}
```

这样做的好处是每一层只暴露自己这一层的错误语义：

- state 层知道 state 错误。
- gRPC 层知道 status code。
- auth 和 presence 层知道自己的业务错误。
- HTTP 层知道状态码和 JSON 错误响应。

## HTTP And gRPC Boundary

当前约定：

- 客户端认证、玩家资料查询和 WebSocket 建连走 logic-server。
- state 数据操作走 state gRPC。
- logic 层依赖 `statecontract.Client` / `PresenceClient`，不直接依赖 generated protobuf client。
- 后续匹配、调度、游戏服务之间的通信可以继续使用 protobuf/gRPC。

HTTP 不应该继续膨胀成所有功能入口。新增好友、在线状态查询、匹配、自定义 lobby 等 logic 功能时，可以先在 logic-server 暴露必要 HTTP 接口，但数据读写仍然应该通过 state client 进入 state-server。

## Removed Old Design

这次重构删除或替换了旧设计：

- 删除旧 Go `net/rpc` 契约占位和 client/server 适配层。
- 使用 `proto/state/v1/state.proto` 和 `internal/contract/statepb` 作为当前 gRPC 传输契约。
- 使用 `internal/state/grpcclient` 和 `internal/state/grpcserver` 替代旧传输层。
- `scripts/run.sh` 从单 logic-server 启动变成 state-server + 两个 logic-server + 可选 nginx。

更早之前删除的旧单进程结构仍然保持删除状态：

- `cmd/game-server`
- 旧 `internal/httpapi`
- 旧 logic 层 Redis repository
- 旧 `room` HTTP/domain 模块
- 旧 Redis transaction helper

`room` 这个命名之前容易和游戏内房间混淆。后续游戏外玩家自定义开局入口更适合叫 `lobby`；真正游戏内承载可以再根据设计命名为 `game session`、`world` 或 `battle`。当前代码先不提前确定这些模块。
