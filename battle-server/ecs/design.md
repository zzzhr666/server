# 战斗服 ECS 设计

## 目标

第一版局内玩法先做一个很小的多人移动 demo，不直接做完整的类 Hades 玩法。

目标流程：

```text
匹配成功的玩家进入 battle room
-> 每个玩家拥有一个角色实体
-> 客户端发送移动输入
-> 服务端按 tick 推进房间世界
-> 服务端广播实体状态快照
-> demo 结束后玩家回到大厅
```

这个版本要足够简单，但结构上要方便后续扩展怪物、攻击、技能、投射物、碰撞、伤害、死亡、掉落和房间通关规则。

## ECS 存储选型

使用整数实体 ID 加 sparse-set 风格的组件池。

核心组件不要用 `unordered_map<Entity, Component>` 存储。哈希表可以用于业务索引，例如 `player_id -> Entity`，但组件数据应该尽量放在连续数组中，方便系统遍历和 CPU cache 命中。

每种组件一个独立组件池：

```text
ComponentPool<Transform>
ComponentPool<Velocity>
ComponentPool<PlayerController>
ComponentPool<MoveInput>
ComponentPool<CharacterStats>
ComponentPool<Health>
```

每个组件池内部维护：

```text
dense_entities[]     拥有该组件的实体 ID
dense_components[]   组件数据，索引与 dense_entities 对齐
sparse[]             entity ID -> dense index
```

删除组件时使用 swap-remove，让 dense 数组保持紧凑。

## 初始组件

```text
Transform
  x, y
  facing_x, facing_y

Velocity
  x, y

MoveInput
  x, y

PlayerController
  player_id

CharacterStats
  move_speed

Health
  current, max
```

`Health` 在第一版移动 demo 中可以暂时不用，但它是后续类 Hades 玩法的基础组件，提前放进设计里。

## 初始系统

第一版只做这些系统：

```text
InputSystem
  MoveInput + CharacterStats -> Velocity

MovementSystem
  Transform + Velocity -> 更新 Transform

SnapshotSystem
  Transform + 身份组件 -> 网络状态快照
```

后续可以按固定 tick 顺序增加：

```text
Input
AI
Skill
Movement
Collision
Combat
Death
Drop
RoomProgress
Snapshot
```

第一版不需要做通用 system scheduler。清晰固定的执行顺序就够了。

## Runtime 边界

`UdpServer` 只负责收包、解包、发包和调用 runtime 入口，不写游戏规则。

`BattleRuntime` 管理正在运行的 battle room。

每个 battle room 拥有一个 `World`，并提供：

```text
start()
receive_input(player_id, input)
tick(dt)
snapshot()
finished()
```

这样可以把网络层、房间生命周期和 ECS 局内玩法分开，后续扩玩法时不需要推倒重来。
