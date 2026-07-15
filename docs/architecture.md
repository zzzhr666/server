# 架构说明

本文档记录当前项目的模块边界、Redis 数据结构和后续扩展方向。README 只保留项目入口信息，详细设计放在这里维护。

## 当前状态

- 语言: Go
- 协议: HTTP + JSON
- 存储: Redis
- 当前模块: `config`、`httpapi`、`player`、`room`、`redisdb`
- 当前功能: 玩家创建/查询、房间创建/查询/列表、加入/离开房间、健康检查

## 模块架构

当前依赖关系:

```text
Client
  |
  v
internal/httpapi
  |
  +--> player.Service
  |      |
  |      v
  |   player.Repository
  |
  +--> room.Service
         |
         +--> room.Repository
         |
         +--> player.Repository
```

底层存储:

```text
player.RedisRepository
room.RedisRepository
  |
  v
redis.Client
```

`room.Service` 依赖 `player.Repository` 来判断玩家是否存在，但不会直接调用玩家 HTTP 或玩家 service。这样房间模块只依赖玩家数据能力，不和玩家业务流程耦合。

## 配置

项目级默认配置位于 `internal/config`:

```text
HTTPAddr: :8080
Redis:
  Addr: 127.0.0.1:6379
  Password: ""
  DB: 0
```

`cmd/game-server/main.go` 只负责读取配置、创建 Redis client、组装 service 和启动 HTTP server。后续可以在 `internal/config` 中增加环境变量或配置文件加载逻辑。

## 模块约定

新增业务模块时优先沿用当前结构:

```text
internal/{module}/
├── model.go
├── errors.go
├── service.go
└── redis_repository.go
```

职责划分:

- `model.go`: 定义领域对象。
- `errors.go`: 定义业务错误。
- `service.go`: 定义 service 接口和业务规则。
- `redis_repository.go`: 封装 Redis key、Redis 命令和数据转换。

依赖方向:

```text
httpapi -> service -> repository -> redisdb/go-redis
```

约束:

- HTTP 层只做协议适配，不直接写业务规则。
- Service 层承载业务规则，不直接暴露 Redis 命令。
- Repository 层封装存储细节，不引用 HTTP request/response。
- 新模块独立建包，例如 `auth`、`friend`、`match`、`battle`。

## Redis 数据结构

当前 key 设计:

```text
game:next_player_id              string, INCR 生成玩家 ID
game:player:{player_id}          hash, 玩家基础信息

game:next_room_id                string, INCR 生成房间 ID
game:room:{room_id}              hash, 房间基础信息
game:room:{room_id}:players      set, 房间成员玩家 ID
game:rooms                       set, 所有房间 ID
```

当前使用的数据类型:

- `String`: 自增 ID 计数器。
- `Hash`: 保存玩家、房间对象字段。
- `Set`: 保存房间成员和房间 ID 集合。

后续可扩展:

```text
game:room:{room_id}:ready_players
game:session:{token}
game:player:{player_id}:friends
game:match:queue:{mode}
game:leaderboard:{mode}
```

## 演进路线

### V1: HTTP 游戏外逻辑

- 完成玩家、房间基础接口
- 增加房间状态、最大人数、准备状态、开始游戏
- 统一错误码和响应结构
- 增加配置文件和环境变量

### V2: 登录、好友、匹配

- `internal/auth`: 游客登录、token、会话
- `internal/friend`: 好友申请、好友列表
- `internal/match`: 匹配队列、自动创建内部房间

### V3: Nginx 负载均衡

- 多开 `game-server` 实例
- 使用 Nginx upstream 做负载均衡
- 通过 Redis 共享玩家、房间和匹配状态
- 使用 `/health` 做探活

### V4: RPC / gRPC 拆分

- 将 Account/Friend/Room/Match 模块拆成独立服务
- HTTP API 服务变成网关层
- 使用 Protobuf 定义内部服务契约
- 将 repository 或 service 实现替换为 RPC/gRPC client

### V5: 游戏内玩法与 ECS

- 新增 `internal/battle`
- 将一局游戏抽象为 GameSession
- 使用 ECS 管理实体、组件和系统
- 将 CPU 密集逻辑保留为可迁移到 C++ 服务的边界

## 近期建议

优先增强房间模块:

- 增加 `status`: `waiting` / `playing` / `closed`
- 增加 `max_players`
- 增加准备状态: `game:room:{id}:ready_players`
- 增加开始游戏接口: `POST /rooms/{id}/start`
