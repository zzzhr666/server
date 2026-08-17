# 战斗 ECS 设计

`battle-server/ecs` 是局内模拟层。它不处理 HTTP、TCP、玩家账号或 Redis；`runtime` 将网络输入转为 `PlayerCommand`，驱动 `World::tick`，再将 snapshot 转为 UDP protobuf。

聊天、好友和实时投递属于局外 Go 服务边界，不进入 ECS 或 battle runtime。聊天历史由 state-server 写入 MongoDB；logic-server 通过 TCP protobuf 提供聊天请求，并消费 `RealtimeDelivery` 后向客户端推送 `chat_message_pushed`。因此战斗房间、ECS 世界和聊天频道彼此独立，战斗断线或房间销毁不会改变聊天历史的 TTL 与分页规则。

## 核心边界

![ECS 核心边界](../../docs/diagrams/ecs-boundary.svg)

- `net` 负责包收发与编解码。
- `session` 校验 player、room、token 并维护 UDP endpoint、conversation 和连接状态。
- `runtime` 维护房间 tick、广播、结束与 rcenter 回调。
- `BattleInstance` 持有一局的 World、波次、经验、祝福候选和结算统计。
- `World` 只处理实体、组件、系统和事件。

## 实体与组件

实体使用整数 ID。组件池采用 sparse-set 风格的紧凑存储：每个池维护实体 dense 数组、组件 dense 数组和 entity-to-index sparse 索引；删除使用 swap-remove。

### 角色与移动

| 组件 | 主要字段 | 用途 |
| --- | --- | --- |
| `Transform` | position、direction | 位置和朝向 |
| `Velocity` | x、y | 当前移动速度 |
| `MoveRequest`、`MoveIntent` | x、y | 客户端移动请求与归一化后的意图 |
| `CharacterStats` | move_speed | 移速 |
| `PlayerController` | 标记组件 | 玩家实体身份 |
| `MonsterController`、`MonsterIdentity` | kind | 怪物身份和种类 |
| `KitingAI` | retreat_distance | 远程怪物的拉开距离规则 |

### 战斗

| 组件 | 主要字段 | 用途 |
| --- | --- | --- |
| `Health` | current_health、max_health | 生命和死亡判定 |
| `AttackDefinition` | kind、damage、range、cooldown、projectile 参数 | 武器或怪物基础攻击 |
| `AttackRequest`、`AttackIntent` | requested、active、damage、context | 输入与本次攻击上下文 |
| `AttackCooldown` | remaining_seconds | 攻击冷却 |
| `Dash`、`DashIntent`、`DashCooldown` | 倍率、剩余时间 | 冲刺规则 |
| `Projectile` | damage、distance、hit_radius、context | 投射物状态 |
| `StatusEffects` | burns、freeze | 燃烧和冰冻状态 |

攻击上下文携带 owner、emitter、action state 和 effect ID，用于保证一轮攻击内的闪电链等 proc 不重复触发。

### 成长与祝福

| 组件 | 用途 |
| --- | --- |
| `PlayerProgress` | 局内等级、经验、升级阈值、未选择的祝福次数 |
| `BlessingInventory` | 已持有祝福和等级 |
| `StatusEffects` | 燃烧、冰冻等由祝福触发的持续状态 |

局外成长不是 ECS 组件。`BattleInstance` 在创建玩家前对武器基础属性调用 `apply_growth`，然后将结果写入角色的 `Health`、`CharacterStats` 和 `AttackDefinition`。

## 固定系统顺序

`World` 构造时注册以下系统，顺序不可随意调换：

```text
move_resolve_system
status_effect_system
monster_ai_system
dash_resolve_system
dash_system
move_system
projectile_hit_system
projectile_range_system
attack_resolve_system
projectile_spawn_system
hit_resolve_system
damage_modify_system
pre_damage_blessing_system
damage_system
blessing_trigger_system
death_system
```

冲刺优先沿当前移动输入方向施加；玩家没有移动输入时，则沿 `Transform.direction` 保存的最后朝向施加。

![战斗处理流水线](../../docs/diagrams/combat-pipeline.svg)

关键规则：

- `status_effect_system` 在本 tick 早期推进燃烧和冰冻；冰冻会影响后续怪物 AI 和攻击。
- `damage_modify_system` 处理暴击；`pre_damage_blessing_system` 在伤害落地前生成闪电链伤害事件。
- `damage_system` 产生实际伤害和 `DamageAppliedEvent`；随后 `blessing_trigger_system` 处理吸血、燃烧和冰冻。
- `death_system` 消费致死事件，移除实体并生成 kill/death 事件。

## 伤害与事件

伤害以事件形式在系统间传递：

```text
AttackIntent / Projectile
  -> DamageEvent
  -> modified_damage
  -> DamageAppliedEvent
  -> KillEvent / DeathEvent
```

`DamageSourceKind` 区分普通攻击、燃烧和闪电链。吸血只响应攻击伤害。燃烧命中会追加独立 `BurnStatus`，因此同一目标允许按当前机制叠层。冻结保留较长的现有剩余时间。

## 波次、升级与祝福

`BattleInstance` 在 `Fighting` 阶段推进 World 并消费 kill events；击杀按怪物种类授予经验。升级会增加 `pending_upgrade_choices`，随后进入 `RewardSelection`：

![奖励选择状态](../../docs/diagrams/reward-selection-state.svg)

每名需要选择的玩家会得到 3 个不同祝福候选。候选被选择后，`BlessingInventory` 中对应祝福新增或升一级。祝福状态与候选会被写入 `WorldSnapshot`，客户端据此显示数值说明和选择 UI。

当前祝福配置在 `ecs/system/blessing_config.hpp`：

| 祝福 | 1 级 | 每级增加 |
| --- | --- | --- |
| 暴击 | 15% 概率，175% 伤害 | 概率 +4%，伤害 +15% |
| 吸血 | 8% | +2% |
| 燃烧 | 6 伤害/秒，2.5 秒 | +2 伤害/秒，+0.5 秒 |
| 冰冻 | 15% 概率，1.0 秒 | 概率 +4%，时长 +0.15 秒 |
| 闪电链 | 50% 原伤害、1 个额外目标、距离 9 | 伤害 +10%，额外目标 +1 |

## Snapshot 与客户端同步

`WorldSnapshot` 包含实体位置、方向、生命、实体类型、怪物类型、当前波次、战斗阶段、奖励选择剩余时间、玩家经验、祝福状态和战斗事件。`BattleInstance` 为攻击和死亡事件保留 60 个 server tick，避免客户端因单个 UDP 快照丢失而完全错过表现事件。

快照不是指令日志，客户端应把服务器快照作为权威状态；本地输入预测或插值不得改变服务器结算结果。

## 测试约定

- `ecs/tests/world_test.cpp` 覆盖实体创建、系统规则、状态效果、伤害和祝福数值。
- `runtime/tests/battle_instance_test.cpp` 覆盖波次、经验、奖励选择和结算。
- `runtime/tests/battle_runtime_test.cpp` 覆盖 room 生命周期、断线 session、快照广播和无人房间超时。

新增组件或系统时，先确定它在上述顺序中的输入和输出事件，再添加局部规则测试；改变系统顺序时必须覆盖原有相互依赖的行为。
