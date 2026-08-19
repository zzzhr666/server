# 架构文档

项目由局外 Go 服务与局内 C++ 战斗服组成。局外侧负责长期玩家状态和对局调度；局内侧负责短生命周期的房间、UDP 会话和确定的 ECS tick。

## 运行拓扑

![运行拓扑](diagrams/runtime-topology.svg)

## 服务职责与状态归属

| 服务 | 职责 | 持久状态 |
| --- | --- | --- |
| `logic-server` | HTTP 健康检查与注册登录、原生 TCP 大厅协议、认证、好友、在线状态、成长、聊天、排行榜和匹配请求适配；维护本实例 TCP 连接并消费实时投递 | 无业务持久状态；连接仅在本实例内存中 |
| `state-server` | 将局外领域操作映射到持久化存储；提供排行榜、聊天历史和实时投递的 gRPC 接口 | Redis 保存局外状态、排行榜与实时事件；MongoDB 保存聊天消息 |
| `rcenter-server` | battle 节点注册、单人建房、双人 FIFO 匹配、活跃对局、奖励和排行榜结算 | 节点、队列、活跃对局在内存；进程重启会丢失这些状态 |
| `battle-server` | 房间、UDP session、tick、ECS、快照和战斗结束 | 房间和世界在内存；不负责账号和好友 |
| Redis | 账号、玩家、session、成长、金币、好友、在线状态、排行榜、实时事件 | `game:*` 键 |
| MongoDB | 世界频道和好友私聊的消息历史 | `chat_messages` 集合；TTL、频道分页和客户端幂等索引；`game-mongo-data` Docker volume |

边界规则：

- `logic-server` 不直接读写 Redis 或 MongoDB，只调用 state gRPC 和 rcenter gRPC。
- rcenter 不解析 UDP 包，也不运行 ECS。
- `battle-server/net` 不承载玩法规则；网络事件交给 `session` 和 `runtime`。
- protobuf 源码只在 `proto/`；Go 和 C++ 生成代码由脚本生成。

## 局外逻辑

### 账号、社交与成长

![局外逻辑](diagrams/out-of-battle-flow.svg)

局外成长初始等级均为 1，最高 10：攻击每级 `+8%`、攻速每级 `+6%`、生命每级 `+10%`、移速每级 `+3%`。rcenter 在创建房间前读取所有匹配玩家的成长等级，并连同英雄对应的战斗配置一起传给 battle-server。跨服务协议为兼容旧客户端仍使用字段名 `hero`。

成长升级价格由 `internal/logic/growth` 配置，TCP `growth_get` 和 `growth_upgrade` 成功响应均返回每项的当前等级、下一级价格和上限。客户端不应自行推导价格。

### TCP、在线状态与匹配

![匹配流程](diagrams/match-flow.svg)

`rcenter-server` 根据 `match_start.solo` 选择模式：单人请求跳过队列并立即创建仅包含当前玩家的房间；双人请求将第一名玩家放入 FIFO 队列，第二名玩家到来时与队头组成房间。节点选择会分别预留 1 个或 2 个玩家容量。成功创建房间后，为每个玩家写入同一份 `ActiveMatch`，其中包含 `room_name`、token、battle 节点、UDP 地址、玩家列表和局外 loadout。未携带 `solo` 的旧客户端按双人模式处理。

客户端仅通过 HTTP 注册或登录并取得 session token。logic TCP 完成认证后承载玩家、成长、好友、在线状态、聊天、排行榜和匹配的全部大厅请求，并自动调用 `ResumeMatch`。客户端也可显式发送 `match_resume`，用于从战斗主动返回大厅后重新进入旧对局。没有活跃对局时，恢复会返回 `active match not found`；客户端应清除本地旧 match 并恢复正常匹配。

同一玩家建立新的 TCP 连接时，logic 会替换旧连接并向旧连接推送 `connection_replaced`。每个 logic 实例启动两个 state 实时订阅：`RealtimeRoute{type=server, server_name=<logic 名称>}` 和 `RealtimeRoute{type=broadcast}`。好友上线、下线、申请、申请处理和删除好友等定向事件投递到目标玩家所在实例；目标玩家连接在其他 logic 实例时，通过 state 的 `RealtimeDelivery` 转发到对应实例。

### 聊天与实时投递

聊天服务位于 `internal/logic/chat`，负责校验内容、好友关系、频道键和分页参数；state-server 的 Mongo store 负责消息写入、幂等去重、容量裁剪、TTL 和历史查询。世界频道使用固定键 `world:global`，发送成功后发布 `broadcast` 路由；各 logic 实例将事件广播给本机连接并排除发送者。私聊频道键按两个玩家 ID 排序生成，发送成功后根据 presence 查找接收者所在 logic 实例并发布 `server` 路由。

实时消息使用 `RealtimeDelivery` 包装明确的 `RealtimeRoute` 和 `RealtimeEvent`。聊天事件包含完整的 `RealtimeChatMessage`，包括 `sender_nickname`，因此客户端可直接渲染昵称。历史查询返回按时间升序排列的一页，`before_message_key` 以本页最早消息为游标读取更早数据；默认页大小为 25，单次最多 100 条。世界频道最多保留 100 条，私聊每个会话最多保留 50 条，最长可读时间均为 24 小时。

### 排行榜与结算

排行榜读取链路为 `realtime Handler -> internal/logic/leaderboard -> state gRPC -> Redis store`。logic 层负责默认 `limit=20`、最大 100、地图版本必填等业务校验；state-server 解析 Redis member，并通过一次 pipeline 批量补全玩家昵称与头像。当前固定地图版本为 `wave-v1`。

排行榜使用三个 Redis ZSET：

| 键 | member | score | 排序 |
| --- | --- | --- | --- |
| `game:leaderboard:clear_time:solo:{map_version}` | 玩家 ID | 最短纯战斗毫秒数 | 升序 |
| `game:leaderboard:clear_time:duo:{map_version}` | 按升序拼接的 `小ID:大ID` | 队伍最短纯战斗毫秒数 | 升序 |
| `game:leaderboard:total_kills` | 玩家 ID | 跨模式累计击杀数 | 降序 |

battle-server 只在 `Fighting` 阶段累加 `combat_duration_ms`，因此 `RewardSelection` 中玩家选择祝福或等待超时的时间不会影响通关排名。结束时该时长随 `FinishMatch` 上报给 rcenter。rcenter 根据一人或两人 roster 写入 `solo` 或 `duo`，胜利局标记为 cleared；当前不接受超过两人的排行榜结算。

state Redis store 在金币结算的同一个乐观事务中使用 `ZADD LT` 保存更短的通关时间，并用 `ZINCRBY` 累加每名玩家的总击杀。失败局不更新通关时间，但正常胜利和失败都会累加击杀。房间名作为 `settlement_id`，幂等标记保留 7 天，保证 battle-server 重试不会重复发放金币或更新排行榜。

## 局内逻辑

### 进入房间与战斗数据流

![战斗数据流](diagrams/battle-data-flow.svg)

`CreateRoom` 先在 battle-server 建立允许玩家列表和 token。客户端发送 `ClientHello` 后，`SessionManager` 绑定 player、UDP conversation 和 endpoint。房间第一次所有玩家都加入时启动 `BattleInstance`；已存在的玩家使用新的 UDP conversation 或 endpoint hello 时会重绑为同一 session，而不是创建第二个玩家。

客户端的 `ClientInput` 包含移动、攻击和冲刺请求。客户端应每 5 秒发送 `ClientHeartbeat`。battle-server 默认 60 tick/s，并向所有已连接 session 广播 `WorldSnapshot`。每个玩家的 `EntitySnapshot.hero` 都携带其英雄映射值，所有客户端据此渲染对应英雄；该值在房间创建时冻结，断线重连不会改变。

### ECS tick

`World` 使用固定顺序调度系统。顺序是游戏规则的一部分：

![ECS tick 顺序](diagrams/ecs-tick-order.svg)

系统结果包括：

- 移动、怪物 AI、冲刺、近战和投射物。
- 命中、伤害、死亡、击杀统计和战斗事件。
- 波次生成与玩家经验；默认 10 波怪物总数依次为 7、8、10、11、13、14、16、17、19、20，最后一波由 12 个近战和 8 个远程怪物组成；每次升级生成 3 个祝福候选。
- 奖励选择阶段暂停正常战斗和排行榜计时，所有待选祝福完成或超时自动选择后进入下一波。

快照包含实体位置和生命、波次、阶段、玩家经验、已持有祝福、待选祝福以及短期保留的攻击/死亡事件。

### 英雄选择、战斗配置与祝福

玩家对外选择初始英雄。服务端使用 `hero` 字段传输英雄对应的攻击配置；战斗服在创建玩家实体前先应用该配置，再应用局外成长。当前配置：

玩家基础移速为 11.0，初始最大生命保持 1000。

| 初始英雄 | 协议值 | 类型 | 伤害 | 攻击间隔 | 范围 | 附加参数 |
| --- | --- | --- | ---: | ---: | ---: | --- |
| Fire | `fire` | 近战 | 23 | 0.34 秒 | 3.0 | - |
| Ice | `ice` | 近战 | 13 | 0.20 秒 | 2.4 | - |
| Rock | `rock` | 近战 | 38 | 0.62 秒 | 3.5 | - |
| Nature | `nature` | 投射物 | 32 | 0.30 秒 | 15.0 | 弹速 25，命中半径 0.85 |

当前祝福及每级数值：

| 祝福 | 1 级 | 每级增加 |
| --- | --- | --- |
| 暴击 | 15% 概率，175% 伤害 | 概率 +4%，伤害 +15% |
| 吸血 | 攻击伤害的 8% | +2% |
| 燃烧 | 6 伤害/秒，2.5 秒 | +2 伤害/秒，+0.5 秒 |
| 冰冻 | 15% 概率，1.0 秒 | 概率 +4%，时长 +0.15 秒 |
| 闪电链 | 50% 原伤害、1 个额外目标、距离 9 | 伤害 +10%，额外目标 +1 |

燃烧命中会按现有机制叠加独立状态；数值调整不改变该机制。

## 断线、恢复与房间销毁

![断线与恢复状态](diagrams/reconnect-state.svg)

`BattleRuntime` 每个 tick 先标记过期 session，再检查房间：

1. 只向 `Connected` session 发送快照和 game over 包。
2. 房间完整 roster 仍保留，用于结束时释放所有玩家，即使部分玩家已断开。
3. 若所有玩家断开，记录首次无人连接的时间。
4. 任一玩家重新 hello 后，清除该房间的无人连接计时。
5. 计时达到 `all_players_disconnected_timeout`（默认 90 秒）时，以 `all_players_disconnected` 结束房间。

战斗结束时，battle-server 调用 rcenter `FinishMatch`，并携带稳定的 `room_name`、完整玩家统计和纯战斗时长。胜利/失败必须携带完整玩家统计；battle-server
对瞬时 RPC 错误执行有限重试，每次默认使用 3 秒 deadline。节点启动注册默认使用独立的 15 秒 deadline，为容器 DNS 与 gRPC
冷连接预留时间；两项超时均可通过 battle-server 命令行参数调整。rcenter 再通过 state-server 以 Redis
乐观事务一次发放全部奖励并更新排行榜，用房间名保证重复回调不会重复到账或累计击杀。只有结算成功或已结算后才释放属于该房间的 `ActiveMatch` 与
in-game 标记，迟到的旧房间回调不会清理玩家的新对局。`all_players_disconnected` 等非胜负原因不携带奖励统计，只执行房间感知的状态释放。

默认金币结算为每局基础 50、胜利额外 100、近战怪物每只 2、远程怪物每只 3，未知怪物类型按每只 2 结算。按默认 10 波全部由单人击杀计算，一次胜利共获得 460 金币；局外成长升级价格保持不变。

## 代码导航

| 问题 | 首选位置 |
| --- | --- |
| HTTP 或 TCP 协议 | `internal/logic/httpapi`、`internal/logic/realtime`、`docs/api.md` |
| 匹配、恢复或奖励 | `internal/rcenter`、`internal/logic/match` |
| 排行榜查询与存储 | `internal/logic/leaderboard`、`internal/state/redisstore` |
| 局外持久状态 | `internal/state`、`internal/logic/*/*repository.go` |
| Room 与 UDP session | `battle-server/game`、`battle-server/session`、`battle-server/net` |
| Room 生命周期和结束 | `battle-server/runtime` |
| 英雄、成长、波次 | `battle-server/gameplay` |
| 局内实体和规则 | `battle-server/ecs`、`battle-server/ecs/design.md` |

## 可观测性

Go 服务和 Battle 服务都提供独立的 Prometheus `/metrics` 端点，由 Compose 中的 Prometheus 统一抓取。Go 侧默认暴露 Go runtime、进程资源和局外领域指标；Battle 侧暴露房间、UDP 会话、控制面请求、UDP 收发、tick 耗时和 tick 超时指标。指标只使用稳定的服务、实例、操作和结果维度，不使用玩家 ID、房间名等高基数标签。详细指标定义与 PromQL 示例见 [metrics.md](metrics.md)。
