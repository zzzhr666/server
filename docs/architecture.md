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

局外成长初始等级均为 1，最高 10：攻击每级 `+8%`、攻速每级 `+4%`、生命每级 `+10%`、移速每级 `+3%`。rcenter 在创建房间前读取所有匹配玩家的英雄和成长等级，并连同 `PlayerLoadout` 传给 battle-server。协议字段仍使用 `hero`。

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

- 移动、怪物 AI、冲刺、近战、投射物和静态障碍物碰撞。
- 尖刺、毒池和沼泽的范围检测；玩家与怪物遵循同一套陷阱规则。
- 命中、伤害、死亡、击杀统计和战斗事件。尖刺和毒池伤害进入统一伤害与死亡链路。
- 玩家经验和祝福选择；祝福选择阶段暂停 World tick，仅推进选择倒计时，不计入排行榜战斗时长。

当前版本已经打通战斗房、奖励房和 Boss 房流程。战斗房按进入、战斗、经验升级、祝福选择、出口选择和切房推进；奖励房从 `EnteringRoom` 进入独立的 `Rewarding` 阶段，每名玩家完成一次免费奖励后进入 `ChoosingExit`；Boss 房只生成 Boss，死亡后直接胜利结算。祝福选择负责把升级次数转换为祝福战力，不属于奖励房免费奖励。

`RoomLayoutCatalog` 提供出生点、怪物、障碍物和陷阱布局，`RoomRuntime` 根据房间类型进入 `Fighting` 或 `Rewarding`，并管理清空、祝福选择、出口选择和切房阶段。`BattleInstance` 负责在切房时统一销毁并重建静态实体；重建失败时会恢复旧地图、静态实体和玩家位置。奖励房不生成怪物，布局中的水泉和商店障碍物分别通过 `reward_fountain`、`shop` 标识。障碍物阻挡角色移动，陷阱保持可通行；陷阱系统在角色移动后检测新位置。

`Rewarding` 和 `ChoosingExit` 仍推进 World，因此客户端可继续发送移动、攻击和冲刺输入；这两个阶段当前没有怪物，也不累计只针对 `Fighting` 的排行榜战斗时长。免费奖励只能在 `Rewarding` 选择一次，可恢复 50% 最大生命、增加 20 点攻击、增加 20 点护甲、随机获得或升级一个祝福，也可跳过。所有玩家完成选择后进入出口选择。

怪物定义携带灵魂奖励：近战怪物 10，远程怪物 5。灵魂只保存在当局 `BattleInstance` 中，用于奖励房商店。商店在 `Rewarding` 和 `ChoosingExit` 均可购买，每名玩家对每件装备限购一次，不同玩家之间不共享库存；购买结果直接修改当局生命上限、攻击、护甲或移速。

尖刺只在目标进入范围时造成一次伤害，离开后重新进入可再次触发。毒池对目标维护单个持续伤害状态，持续停留只刷新有效期，不叠层也不重置下一次伤害的计时。沼泽降低玩家普通移动以及怪物追击、撤退速度，但玩家冲刺不受减速影响。

快照包含实体位置、生命、权威碰撞半径、实体类型、场景对象类型、房间流程、玩家经验、已持有祝福、待选祝福以及短期保留的攻击/死亡事件。玩家灵魂和战斗属性在所有阶段发送；战斗属性包括攻击伤害、移速、攻击间隔和护甲。免费奖励完成状态只在 `Rewarding` 发送，商店报价、装备定义和个人购买记录只在奖励房的 `Rewarding`、`ChoosingExit` 发送。

### 英雄选择、战斗配置与祝福

玩家对外选择初始英雄。服务端使用 `hero` 字段传输英雄对应的攻击配置；战斗服在创建玩家实体前先应用该配置，再应用局外成长。当前配置：

玩家基础移速为 11.0，初始最大生命为 500。局内基础数值统一定义在 `battle-server/gameplay/gameplay_config.hpp`；`blessing_config.hpp` 只保留按祝福等级计算最终值的公式，不保存另一套基础数值。

| 初始英雄 | 协议值 | 类型 | 伤害 | 攻击间隔 | 范围 | 附加参数 |
| --- | --- | --- | ---: | ---: | ---: | --- |
| Fire | `fire` | 近战 | 21 | 0.43 秒 | 3.0 | - |
| Ice | `ice` | 近战 | 12 | 0.28 秒 | 2.4 | - |
| Rock | `rock` | 近战 | 34 | 0.78 秒 | 3.5 | - |
| Nature | `nature` | 投射物 | 29 | 0.38 秒 | 15.0 | 弹速 25，命中半径 0.85 |

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
5. session 连续 `session_idle_timeout`（默认 10 秒）无有效输入或心跳后标记断开；计时达到 `all_players_disconnected_timeout`（默认 90 秒）时，以 `all_players_disconnected` 结束房间。

战斗结束时，battle-server 调用 rcenter `FinishMatch`，并携带稳定的 `room_name`、完整玩家统计和纯战斗时长。胜利/失败必须携带完整玩家统计；battle-server
对瞬时 RPC 错误执行有限重试，每次默认使用 3 秒 deadline。节点启动注册默认使用独立的 15 秒 deadline，为容器 DNS 与 gRPC
冷连接预留时间；两项超时均可通过 battle-server 命令行参数调整。rcenter 再通过 state-server 以 Redis
乐观事务一次发放全部奖励并更新排行榜，用房间名保证重复回调不会重复到账或累计击杀。只有结算成功或已结算后才释放属于该房间的 `ActiveMatch` 与
in-game 标记，迟到的旧房间回调不会清理玩家的新对局。`all_players_disconnected` 等非胜负原因不携带奖励统计，只执行房间感知的状态释放。

默认金币结算为每局基础 50、胜利额外 100、近战怪物每只 2、远程怪物每只 3，未知怪物类型按每只 2 结算。最终奖励按该局实际上报的击杀统计计算；局外成长升级价格保持不变。

## 代码导航

| 问题 | 首选位置 |
| --- | --- |
| HTTP 或 TCP 协议 | `internal/logic/httpapi`、`internal/logic/realtime`、`docs/api.md` |
| 匹配、恢复或奖励 | `internal/rcenter`、`internal/logic/match` |
| 排行榜查询与存储 | `internal/logic/leaderboard`、`internal/state/redisstore` |
| 局外持久状态 | `internal/state`、`internal/logic/*/*repository.go` |
| Room 与 UDP session | `battle-server/game`、`battle-server/session`、`battle-server/net` |
| Room 生命周期和结束 | `battle-server/runtime` |
| 英雄、成长、房间图与布局 | `battle-server/gameplay` |
| 局内实体和规则 | `battle-server/ecs`、`battle-server/ecs/design.md` |

## 可观测性

Go 服务和 Battle 服务都提供独立的 Prometheus `/metrics` 端点，由 Compose 中的 Prometheus 统一抓取。Go 侧默认暴露 Go runtime、进程资源和局外领域指标；Battle 侧暴露房间、UDP 会话、控制面请求、UDP 收发、tick 耗时和 tick 超时指标。指标只使用稳定的服务、实例、操作和结果维度，不使用玩家 ID、房间名等高基数标签。详细指标定义与 PromQL 示例见 [metrics.md](metrics.md)。
