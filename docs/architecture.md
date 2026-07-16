# 架构说明

本文档记录当前项目的模块边界、Redis 数据结构和后续扩展方向。README 只保留项目入口信息，详细设计放在这里维护。

## 当前状态

- 语言: Go
- 协议: HTTP + JSON
- 存储: Redis
- 当前模块: `config`、`httpapi`、`player`、`room`、`redisdb`
- 当前功能: 玩家创建/查询/更新、房间创建/查询/列表、加入/离开房间、准备/取消准备、开始游戏、健康检查

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
game:room:{room_id}:ready_players  set, 已准备玩家 ID
game:rooms                       set, 所有房间 ID
game:player:{player_id}:room     string, 玩家当前所在房间 ID
```

当前使用的数据类型:

- `String`: 自增 ID 计数器。
- `Hash`: 保存玩家、房间对象字段。
- `Set`: 保存房间成员和房间 ID 集合。

后续可扩展:

```text
game:session:{token}
game:player:{player_id}:friends
game:match:queue:{mode}
game:leaderboard:{mode}
```

## 目标架构

培训路线中的最终形态可以拆成四类服务:

```text
Client
  |
  | HTTP 登录、主界面、开始匹配
  v
Nginx
  |
  v
logicserver
  |
  | RPC: 开始匹配、取消匹配、查询匹配
  v
rcenterserver
  |
  | gRPC + Protobuf: 房间分配、负载上报、房间生命周期
  v
roomserver / gameserver
  ^
  |
  | TCP: token 验证后进入游戏
  |
Client
```

核心流程:

1. 玩家通过 Nginx 对外地址访问 `logicserver`，使用用户名、密码或游客信息登录。
2. `logicserver` 完成账号验证，返回会话 token，主界面继续走 HTTP API。
3. 玩家点击开始游戏后，`logicserver` 将玩家信息和匹配参数通过 RPC 发送给 `rcenterserver`。
4. `rcenterserver` 维护匹配队列并运行匹配算法。
5. 匹配成功后，`rcenterserver` 根据负载选择一个 `roomserver`，生成 `room_id` 和一次性入房 `game_token`。
6. `rcenterserver` 将 `game_token`、`room_id`、玩家列表、过期时间等写入 Redis，并把 `roomserver` 地址和 token 返回给 `logicserver`。
7. `logicserver` 把连接信息返回客户端。
8. 客户端使用 TCP 直连 `roomserver`，首次消息携带 `game_token`。
9. `roomserver` 读取 Redis 校验 token，拿到 `room_id` 和玩家信息后加入对应房间并开始游戏。
10. `roomserver` 通过 gRPC 向 `rcenterserver` 周期上报负载，例如在线玩家数、房间数、CPU 估算负载。
11. 游戏结束后，`roomserver` 返回结算结果，断开游戏连接，玩家回到主界面。

## 平滑迁移原则

当前代码已经是一个适合演进的模块化单体。升级时优先遵守以下原则:

- 当前项目先定位为 `logicserver` 的第一版，不急着拆多个进程。
- 先新增模块和接口，再替换实现；不要先搬代码到新服务。
- HTTP 对外接口尽量稳定，客户端不应该感知内部从本地实现换成 RPC/gRPC。
- Redis key 的所有权要逐步明确，多实例前必须把跨请求状态放到 Redis。
- 每次只移动一个边界，例如先把 `match.Service` 抽象出来，再把它的实现换成 `rcenterserver` RPC client。
- 不做一次性大拆包。新增包仍沿用 `internal/{module}/model.go`、`errors.go`、`service.go`、`redis_repository.go`。

## 当前项目定位

当前代码可以视为 `logicserver v0.2`:

- 已具备 `player` 模块: 创建、查询、更新玩家资料。
- 已具备 `room` 模块: 创建、列表、详情、加入、离开、准备、取消准备、开始游戏。
- 房间详情已经能返回房间内玩家状态: `id`、`name`、`avatar`、`ready`、`owner`。
- `RoomService` 使用 `sync.RWMutex` 保护单进程内的多步房间操作。
- Redis 已经保存玩家、房间、房间成员、准备状态和玩家当前房间索引。

这个状态足够作为继续扩展的基线。下一步不建议直接引入 `rcenterserver` 或 `roomserver`，因为登录、会话、在线状态和匹配入口还没有稳定。

## 迭代计划

### Phase 1: 补齐 logicserver 基础能力

目标: 让当前服务成为一个比较完整的游戏外逻辑服务。

新增模块:

```text
internal/auth
internal/presence
internal/friend
```

建议顺序:

1. `auth`: 登录、登出、查询当前玩家。
2. `presence`: 心跳、在线状态、玩家状态查询。
3. `GET /players`: 当前玩家列表，可返回基础资料和在线状态。
4. `friend`: 好友申请、好友列表、删除好友。

建议接口:

```text
POST /auth/login
POST /auth/logout
GET  /me
POST /presence/heartbeat
GET  /players
POST /friends/requests
GET  /friends/requests
POST /friends/requests/{id}/accept
DELETE /friends/{id}
GET  /friends
```

Redis key 建议:

```text
game:account:{username}              hash, 登录账号
game:session:{token}                 hash, token -> player_id / expire_at
game:player:{player_id}:sessions     set, 玩家当前 session
game:presence:{player_id}            string, 在线状态或最后心跳时间
game:player:{player_id}:friends      set, 好友玩家 ID
game:friend_request:{request_id}     hash, 好友申请
game:player:{player_id}:friend_requests  set, 玩家收到的申请 ID
```

实现要点:

- `auth.Service` 只负责认证和 session，不直接处理好友或房间。
- HTTP 层通过 `Authorization: Bearer <token>` 找到当前玩家。
- 第一版可以先做简单密码校验或游客登录，后续再升级密码哈希和账号表。
- 在线状态第一版可以用心跳时间判断，例如最近 30 秒有心跳就是在线。
- `player.Service` 可以新增 `List`，但玩家在线状态建议由 `presence.Service` 组合，而不是塞进玩家基础模型。

保留不变:

- 现有玩家和房间 HTTP 接口先不删除。
- `room.Service` 当前业务规则继续可用。
- `cmd/game-server/main.go` 仍然只做依赖组装。

### Phase 2: 为 Nginx 和多 logicserver 做准备

目标: 同一份 `logicserver` 可以启动多个实例，并通过 Nginx 反向代理。

新增内容:

```text
deploy/nginx.conf
docs/deploy.md
internal/config 环境变量读取
```

部署形态:

```text
Nginx
  |
  +--> logicserver :8081
  +--> logicserver :8082
  |
 Redis
```

需要调整:

- `HTTPAddr` 从环境变量读取，例如 `GAME_HTTP_ADDR=:8081`。
- `/health` 保持轻量探活。
- 所有 session、presence、房间、匹配状态必须在 Redis 中共享，不能依赖进程内内存。

原子性升级点:

- 当前 `sync.RWMutex` 只保证单个进程内的房间操作不交错。
- 多 `logicserver` 之后，不同进程有不同的锁，`sync.RWMutex` 无法保证跨进程原子性。
- 进入多实例前，房间关键操作应迁移为 Redis Lua 或 Redis transaction，例如创建房间、加入房间、离开房间、准备、开始游戏。

建议迁移方式:

1. 保留 `room.Service` 对外方法不变。
2. 在 `room.Repository` 内新增更粗粒度的原子方法，例如 `CreateRoomWithOwner`、`JoinRoomIfAllowed`。
3. 先让 service 调用新 repository 方法。
4. repository 内部用 Lua 完成检查和写入。
5. HTTP 和 service 接口不变，调用方无感知。

这样升级 Redis 原子性不会引发大规模重构。

### Phase 3: 在单体内先加入 match 抽象

目标: 在引入 `rcenterserver` 前，先把匹配入口和匹配状态稳定下来。

新增模块:

```text
internal/match
```

建议接口:

```text
POST   /match/queue
DELETE /match/queue
GET    /match/status
```

本阶段先让 `logicserver` 内部实现 `match.Service`:

```text
httpapi -> match.Service -> match.Repository -> Redis
```

Redis key 建议:

```text
game:match:queue:{mode}              zset/list, 匹配队列
game:match:player:{player_id}        hash, 玩家当前匹配状态
game:match:result:{match_id}         hash, 匹配结果
```

第一版匹配算法:

- 按模式分队列。
- 固定人数凑齐就匹配成功。
- 超时后允许取消或返回等待中。
- 匹配成功后先复用当前 `room.Service.Create/Join` 创建逻辑房间。

为什么先这样做:

- 客户端可以先接入完整的开始游戏流程。
- 匹配状态、错误码、请求/响应结构会先稳定。
- 后续把本地 `match.Service` 换成 RPC client 时，HTTP 层不需要改。

### Phase 4: 引入 rcenterserver

目标: 把匹配队列、匹配算法、roomserver 选择从 `logicserver` 中移出。

新增进程:

```text
cmd/rcenter-server/main.go
internal/rcenter
internal/rcenterclient
```

迁移方式:

1. 在 `logicserver` 中保留 `match.Service` 接口。
2. 新增 `internal/rcenterclient.Client`，实现 `match.Service` 需要的能力。
3. 先用 HTTP 或 Go RPC 跑通内部调用，协议稳定后再考虑 Protobuf。
4. `rcenterserver` 接管 Redis 中的匹配队列和匹配结果。
5. `logicserver` 只负责鉴权、参数校验、调用 rcenter、把结果返回客户端。

服务关系:

```text
Client -> Nginx -> logicserver -> rcenterserver -> Redis
```

`rcenterserver` 职责:

- 加入匹配队列。
- 取消匹配。
- 查询匹配状态。
- 执行匹配算法。
- 管理可用 roomserver 列表和负载。
- 生成 `room_id` 和 `game_token`。
- 写入 token 相关 Redis 数据。

roomserver 选择策略:

第一版使用简单负载分:

```text
score = online_players * 1.0 + running_rooms * 5.0
```

选择 score 最低且最近有心跳的 `roomserver`。后续可以把 CPU、内存、地区、游戏模式权重加入 score，但不要一开始做复杂调度。

Redis key 建议:

```text
game:rcenter:roomservers                  set, roomserver_id 集合
game:roomserver:{server_id}:load          hash, 地址、玩家数、房间数、最后心跳
game:game_token:{token}                   hash, room_id / player_id / match_id / roomserver_id / expire_at
game:game_room:{room_id}                  hash, match_id / roomserver_id / status
game:game_room:{room_id}:players          set, 本局玩家 ID
```

### Phase 5: 引入 roomserver / gameserver

目标: 游戏内连接和一局游戏生命周期从 `logicserver` / `rcenterserver` 中独立出来。

新增进程:

```text
cmd/room-server/main.go
internal/roomserver
proto/roomserver.proto
proto/rcenter.proto
```

连接方式:

```text
Client --TCP--> roomserver
roomserver --gRPC--> rcenterserver
roomserver --Redis--> token / room data
```

roomserver 职责:

- 启动后向 `rcenterserver` 注册自己的 `server_id` 和对外 TCP 地址。
- 周期性上报负载。
- 接收客户端 TCP 连接。
- 校验客户端发送的 `game_token`。
- 根据 Redis 中的 `room_id` 把玩家放入对应房间。
- 运行游戏逻辑。
- 游戏结束后返回结算结果并断开连接。

`rcenterserver` 和 `roomserver` 的 gRPC 建议接口:

```text
RegisterRoomServer(server_id, public_addr)
ReportLoad(server_id, running_rooms, online_players)
CreateGameRoom(room_id, players, game_mode)
CloseGameRoom(room_id, result)
```

第一版 `roomserver` 可以很薄:

- 不实现复杂游戏逻辑。
- 只完成 TCP 建连、token 校验、进入房间、广播一条开始消息、返回模拟结算。
- 先把链路跑通，再增加真实玩法。

### Phase 6: 结算与回主界面

目标: 游戏结束后，服务端能记录结果，客户端能自然回到主界面。

新增模块:

```text
internal/settlement
```

Redis key 建议:

```text
game:result:{game_id}                hash, 一局游戏结果
game:player:{player_id}:recent_games list, 最近对局
```

流程:

1. `roomserver` 计算结果。
2. `roomserver` 写入 Redis 或通过 gRPC 通知 `rcenterserver`。
3. `rcenterserver` 标记房间结束，并释放 roomserver 负载。
4. 客户端收到结算结果，关闭 TCP 连接。
5. 客户端回主界面后继续通过 `logicserver` 查询玩家、好友、在线状态。

## 推荐近期任务

当前最适合继续做的顺序:

1. `auth`: 登录、登出、`GET /me`。
2. `presence`: 心跳和在线状态。
3. `GET /players`: 玩家列表加在线状态。
4. `friend`: 好友申请和好友列表。
5. `match`: 单体内本地匹配队列。
6. Nginx 双实例部署验证。
7. Redis Lua 原子化房间和匹配关键操作。
8. `rcenterserver` 抽离匹配。
9. `roomserver` TCP 入房和 gRPC 负载上报。

每一步都应该保持“先本地接口、再远程实现”的节奏。这样当前代码会自然从单体 `logicserver` 长成多服务架构，而不是中途重写。

## 不建议现在做的事

- 不建议现在把 `player`、`room` 直接拆成多个仓库或多个进程。
- 不建议现在把所有 HTTP 请求改成 Protobuf。
- 不建议在没有 `auth` 和 `presence` 的情况下直接做复杂匹配。
- 不建议在单实例阶段过早引入分布式锁；房间并发先用当前 `sync.RWMutex`，多实例前再迁移 Lua。
- 不建议把在线状态写进 `player.Player` 基础模型，在线状态是运行态数据，应由 `presence` 管理。
