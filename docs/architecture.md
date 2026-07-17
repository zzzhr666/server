# Architecture

项目正在从单进程 HTTP + Redis demo 迁移成多进程游戏服务器 demo。当前已经完成的核心变化是：`logic-server` 不再直接访问 Redis，而是通过 RPC 调用独立的 `state-server`，由 `state-server` 统一操作 Redis。

## Runtime View

当前可运行链路：

```text
Client
  |
  | HTTP JSON
  v
logic-server (:8080)
  |
  | auth/player service
  v
logic state repository adapter
  |
  | statecontract.Client
  v
state rpcclient
  |
  | Go net/rpc, StateService.*
  v
state-server (127.0.0.1:9001)
  |
  | rpcserver adapter
  v
state service
  |
  | accountStore / playerStore / sessionStore
  v
redisstore
  |
  v
Redis (127.0.0.1:6379)
```

这条链路的意义是把“业务入口”和“数据状态操作”拆开：

- `logic-server` 负责 HTTP、登录业务、玩家资料业务。
- `state-server` 负责状态数据读写和跨数据组合操作。
- Redis 只被 `state-server` 直接访问。

## Process Responsibilities

### logic-server

入口：`cmd/logic-server/main.go`

职责：

- 启动 HTTP 服务。
- 注册 `/health` 和 `/auth/*` 路由。
- 创建 `auth.Service` 和 `player.Service`。
- 通过 `rpc.Dial` 连接 `state-server`。
- 使用 `internal/state/rpcclient.Client` 作为 state client。

它依赖 state 契约，但不关心 state 的真实存储是 Redis、MySQL，还是别的服务。

### state-server

入口：`cmd/state-server/main.go`

职责：

- 连接 Redis。
- 创建 Redis store。
- 创建 state service。
- 把 state service 注册成 `net/rpc` 服务。
- 监听 `127.0.0.1:9001`。

所有跨账号、玩家、会话的组合写操作，都应该尽量放在 `state-server` 内部做成一个粗粒度方法，而不是让 `logic-server` 连续调用多个细粒度 RPC。

例如注册账号现在使用：

```text
logic auth service
  -> state.RegisterAccount(...)
  -> state service 内部创建 player、account、session
```

这样比下面这种方式更容易控制并发和一致性：

```text
logic-server
  -> NextPlayerID
  -> CreatePlayer
  -> CreateAccount
  -> CreateSession
```

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

internal/
├── contract/
│   ├── rpc/
│   └── state/
├── logic/
│   ├── auth/
│   ├── player/
│   └── httpapi/
├── platform/
│   ├── config/
│   └── redisdb/
└── state/
    ├── redisstore/
    ├── rpcclient/
    ├── rpcserver/
    └── service/
```

### internal/contract/state

这是 state-server 对外暴露的共享契约。

主要内容：

- `Account`
- `Player`
- `Session`
- `RegisterAccountInput`
- `RegisterAccountResult`
- `Client` 接口
- state 级错误，例如 `ErrAccountExists`、`ErrSessionNotFound`

`logic-server` 依赖这个接口，不依赖 state-server 的具体实现。

### internal/logic/auth

登录业务层。

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

### internal/logic/httpapi

HTTP 适配层。

主要职责：

- 定义 HTTP 路由。
- 解析 JSON 请求。
- 读取 `Authorization: Bearer <token>`。
- 调用 logic service。
- 把业务错误映射为 HTTP 状态码。
- 输出 JSON 响应。

HTTP 层不直接访问 Redis，也不直接调用 state RPC。

### internal/state/service

state 业务层。

主要职责：

- 对 account、player、session 操作加锁。
- 实现跨资源组合操作。
- 调用 store 接口完成实际读写。

当前 store 接口有三类：

- `accountStore`
- `playerStore`
- `sessionStore`

组合操作示例：`RegisterAccount` 会在 state-server 内部完成账号是否存在检查、玩家 ID 生成、玩家创建、账号创建、session 创建。

### internal/state/redisstore

Redis 存储实现。

主要职责：

- 把 state 模型存入 Redis。
- 从 Redis 读取 state 模型。
- 维护 player ID 自增键。

当前 key 大致包括：

```text
game:account:<username>
game:player:<id>
game:session:<token>
game:next_player_id
```

这个包不处理 HTTP，不处理 RPC，也不决定业务流程。

如果以后从 Redis 换成 MySQL，优先新增一个 MySQL store，让它实现 `state/service` 需要的 store 接口。理论上 `logic` 层不应该被影响。

### internal/state/rpcserver

state-server 使用的 RPC 适配层。

主要职责：

- 定义 `net/rpc` 的 args/reply 类型。
- 把 `StateService.Method` 调用转发给 `statecontract.Client`。
- 对外暴露 RPC 服务名 `StateService`。

Go `net/rpc` 要求方法形态接近：

```go
func (s *Server) Method(args Args, reply *Reply) error
```

所以这里会有一些 args/reply 包装类型。

### internal/state/rpcclient

logic-server 使用的 RPC 客户端适配层。

主要职责：

- 持有 `*rpc.Client`。
- 实现 `statecontract.Client` 接口。
- 调用 `StateService.Method`。
- 把 RPC 返回的错误字符串映射回 state sentinel error。

错误映射很重要，因为 Go `net/rpc` 跨进程返回错误时，只能稳定拿到错误文本。客户端需要把这些错误文本重新映射成 `statecontract.ErrAccountExists` 这类哨兵错误，logic 层的 `errors.Is` 才能继续工作。

## Dependency Direction

依赖方向应该保持为：

```text
cmd/logic-server
  -> internal/logic/*
  -> internal/contract/state
  -> internal/state/rpcclient

cmd/state-server
  -> internal/state/rpcserver
  -> internal/state/service
  -> internal/state/redisstore
  -> internal/platform/redisdb
```

不建议出现：

```text
internal/logic/* -> internal/state/redisstore
internal/logic/* -> internal/platform/redisdb
internal/logic/httpapi -> internal/state/rpcclient
```

原因是 logic 的业务代码应该通过接口表达“我要什么状态能力”，而不是知道 Redis 或 RPC 的细节。

## Locking And Atomicity

当前 demo 的并发控制放在 `internal/state/service`。

原则：

- store 只做存储读写，不加业务锁。
- service 对外暴露操作时加锁。
- 跨多个资源的组合操作，在 service 里一次性完成。
- 如果一个组合操作需要多把锁，按固定顺序拿锁。

当前 `RegisterAccount` 的锁顺序是：

```text
accountMu -> playerMu -> sessionMu
```

后续新增组合操作时，应该复用固定顺序，避免死锁。

注意：这只是当前 demo 的进程内并发控制方式。它能约束进入同一个 `state-server` 进程的操作。如果未来 state-server 做多实例部署，需要再引入更完整的分布式锁、数据库事务或单主写入模型。

## Error Flow

以重复注册为例：

```text
redisstore
  -> statecontract.ErrAccountExists
state service
  -> statecontract.ErrAccountExists
rpcserver
  -> error text over net/rpc
rpcclient
  -> map back to statecontract.ErrAccountExists
logic auth state repository
  -> auth.ErrAccountExists
httpapi
  -> HTTP 409 {"error":"account already exists"}
```

这样做的好处是每一层只暴露自己这一层的错误语义：

- state 层知道 state 错误。
- auth 层知道 auth 错误。
- HTTP 层知道状态码和 JSON 错误响应。

## HTTP And RPC Boundary

当前约定：

- 登录相关入口走 HTTP。
- state 数据操作走 RPC。
- 后续匹配、调度、游戏服务之间的通信也倾向走 RPC 或 gRPC。
- 游戏内实时通信协议暂不在当前阶段确定。

HTTP 不应该继续膨胀成所有功能入口。新增好友、在线状态等 logic 功能时，可以先在 logic-server 暴露必要 HTTP 接口，但它们的数据读写仍然应该通过 state client 进入 state-server。

## gRPC Migration Path

当前使用 Go `net/rpc` 是为了先把多进程边界跑通。后续迁移到 protobuf/gRPC 时，主要改动应该集中在：

- 新增 `.proto` 文件，定义 state service 的消息和 RPC 方法。
- 用生成代码替代 `internal/state/rpcserver` 的 net/rpc args/reply。
- 用 gRPC client 替代 `internal/state/rpcclient`。
- 保留或小改 `internal/contract/state` 的业务模型和接口。
- 尽量不改 `internal/logic/auth`、`internal/logic/player`、`internal/state/service`、`internal/state/redisstore` 的核心业务逻辑。

也就是说，当前 `statecontract.Client` 的价值就是隔离传输协议。只要 logic 层依赖接口，不直接依赖 `net/rpc`，后续替换传输层的范围就可控。

## Removed Old Design

这次重构删除了旧的单进程结构：

- `cmd/game-server`
- 旧 `internal/httpapi`
- 旧 logic 层 Redis repository
- 旧 `room` HTTP/domain 模块
- 旧 Redis transaction helper

`room` 这个命名之前容易和游戏内房间混淆。后续游戏外玩家自定义开局入口更适合叫 `lobby`；真正游戏内承载可以再根据设计命名为 `game session`、`world` 或 `battle`。当前代码先不提前确定这些模块。
