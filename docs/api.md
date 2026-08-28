# 接口文档

客户端使用 HTTP `http://localhost:8080` 完成健康检查、注册和登录，并连接原生 TCP 实时服务 `localhost:8081` 完成其余大厅操作。HTTP 响应为 JSON，错误格式为 `{"error":"error message"}`。

## HTTP

### 健康检查与认证

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/health` | 否 | 返回当前 logic-server 名称 |
| `POST` | `/auth/register` | 否 | 注册账号、创建玩家和 session，状态 `201` |
| `POST` | `/auth/login` | 否 | 登录并创建 session |

注册请求：

```json
{
  "username":"alice",
  "pasfire":"pasfire123",
  "nickname":"Alice",
  "avatar":"adventurer",
  "email":"alice@example.com",
  "phone":"13800000000"
}
```

注册与登录成功响应：

```json
{
  "token":"session-token",
  "player":{"id":1,"nickname":"Alice","avatar":"adventurer","email":"alice@example.com","phone":"13800000000","coins":0}
}
```

`avatar` 使用客户端内置资源的稳定标识，可选值为 `adventurer`、`warrior`、`mage`、`priest`、`summoner`；注册时省略或传空值使用默认头像 `adventurer`。

## TCP 实时协议

TCP 服务监听 `:8081`。每条应用帧使用 `GRTP` 格式：4 字节 ASCII magic `GRTP`、1 字节版本 `1`、4 字节大端 payload 长度、4 字节大端 CRC32C（Castagnoli）和 protobuf payload。payload 长度必须大于 0 且不超过 1 MiB；payload 是 `proto/realtime/v1/realtime.proto` 的 `ClientEnvelope` 或 `ServerEnvelope`。

客户端连接后第一帧必须是 `authenticate`，其中 `request_id` 必须非零：

```text
ClientEnvelope { request_id: 1, authenticate: { token: "session-token" } }
ServerEnvelope { request_id: 1, authenticated: { player_id: 7 } }
```

认证成功后服务端自动尝试恢复活跃对局。所有客户端请求都必须携带非零 `request_id` 和一个 payload；客户端请求的响应沿用对应 `request_id`。自动恢复结果、匹配结果和好友事件等服务端主动推送使用 `request_id=0`。

| 客户端 payload | 成功响应 | 行为 |
| --- | --- | --- |
| `heartbeat` | `heartbeat_ack` | 刷新在线状态 |
| `match_start { hero, team_size, solo }` | `match_result` | 开始 1-4 人对局；`team_size=1` 立即创建单人房间，`team_size=2/3/4` 进入对应的独立 FIFO 队列。兼容旧客户端：未设置 `team_size` 时，`solo=true` 按 1 人处理，否则按 2 人处理；`hero` 空值默认 Fire |
| `match_cancel` | `match_canceled` | 取消等待中的匹配 |
| `match_resume` | `match_result` | 请求当前玩家的活跃对局 |
| `player_get` | `player` | 返回当前玩家档案与金币 |
| `player_avatar_update { avatar }` | `player` | 更新预定义头像并返回最新的完整玩家档案 |
| `growth_get` | `growth` | 返回成长等级和每项升级选项 |
| `growth_upgrade { type }` | `growth_upgrade_result` | 扣除金币并升级，返回最新成长、余额和费用 |
| `friend_request_send { to_player_id }` | `friend_request_sent` | 发送好友申请 |
| `friend_request_list_incoming` | `friend_requests` | 返回收到的好友申请 |
| `friend_request_list_outgoing` | `friend_requests` | 返回发出的好友申请 |
| `friend_request_accept { from_player_id }` | `friend_request_handled_ack` | 接受好友申请 |
| `friend_request_reject { from_player_id }` | `friend_request_handled_ack` | 拒绝好友申请 |
| `friend_list` | `friends` | 返回好友档案与在线状态摘要 |
| `friend_delete { friend_player_id }` | `friend_deleted` | 删除好友关系 |
| `logout` | `logout_ack` | 删除当前 session，服务端随后关闭 TCP 连接 |
| `chat_world_send { content, client_message_key }` | `chat_sent` | 写入世界频道并广播给其他在线连接 |
| `chat_direct_send { receiver_id, content, client_message_key }` | `chat_sent` | 向好友发送私聊消息，并主动推送给接收者 |
| `chat_world_list { limit, before_message_key }` | `chat_messages` | 分页读取世界频道历史 |
| `chat_direct_list { friend_id, limit, before_message_key }` | `chat_messages` | 分页读取与好友的私聊历史 |
| `leaderboard_list { type, map_version, limit }` | `leaderboard` | 查询单人通关、双人通关或累计击杀榜 |

匹配和恢复使用 `match_result`，其中包含 `status`、`room_name`、`token`、`battle_node_name` 和 `battle_udp_addr`。单人请求直接返回 `status=matched`；2-4 人请求在对应队列凑齐前返回 `status=waiting`。`status=matched` 后，客户端向 `battle_udp_addr` 发送 UDP `ClientHello(room_name, player_id, token)`。

成长升级的合法 `type` 为 `attack`、`attack_speed`、`health`、`move_speed`。业务请求失败时服务端发送 `error { code, message }` 并保持连接；`code` 取值为 `UNAUTHENTICATED`、`INVALID_ARGUMENT`、`NOT_FOUND`、`CONFLICT` 或 `INTERNAL`。协议帧错误、认证失败、logout 和写入失败会关闭连接。

### 排行榜协议

通关时间榜类型包括 `SOLO_CLEAR_TIME`、`DUO_CLEAR_TIME`、`TRIO_CLEAR_TIME` 和 `QUAD_CLEAR_TIME`；累计击杀使用 `TOTAL_KILLS`。

`leaderboard_list.type` 使用 `LeaderboardType`，支持：

| 类型 | `map_version` | 排序与 `score` | `players` |
| --- | --- | --- | --- |
| `LEADERBOARD_TYPE_SOLO_CLEAR_TIME` | 必填，当前为 `wave-v1` | 纯战斗毫秒数升序 | 1 名玩家 |
| `LEADERBOARD_TYPE_DUO_CLEAR_TIME` | 必填，当前为 `wave-v1` | 队伍纯战斗毫秒数升序 | 组成该队伍的 2 名玩家 |
| `LEADERBOARD_TYPE_TOTAL_KILLS` | 忽略 | 跨单人和双人模式的累计击杀数降序 | 1 名玩家 |

请求的 `limit=0` 使用默认值 20，合法范围为 1 到 100；负数或大于 100 返回 `INVALID_ARGUMENT`。响应会回显规范化后的 `type`、`map_version` 和从 1 开始的 `rank`。玩家信息包含 `player_id`、`nickname` 和 `avatar`；若排行榜中的历史玩家档案已不存在，仍返回其 ID，昵称和头像为空。

通关榜只记录胜利局，并使用玩家或双人队伍在同一地图版本下的历史最短时间。计时只覆盖 battle-server 的 `Fighting` 阶段，祝福选择阶段不计入。累计击杀榜同时统计正常胜利和失败的对局。

示例：

```text
ClientEnvelope {
  request_id: 20
  leaderboard_list {
    type: LEADERBOARD_TYPE_DUO_CLEAR_TIME
    map_version: "wave-v1"
    limit: 20
  }
}
ServerEnvelope {
  request_id: 20
  leaderboard {
    type: LEADERBOARD_TYPE_DUO_CLEAR_TIME
    map_version: "wave-v1"
    entries {
      rank: 1
      players { player_id: 7 nickname: "Alice" avatar: "mage" }
      players { player_id: 8 nickname: "Bob" avatar: "warrior" }
      score: 83250
    }
  }
}
```

### 聊天协议

`chat_world_send` 和 `chat_direct_send` 都必须提供非空 `content` 和客户端生成的幂等键 `client_message_key`。私聊仅允许已经建立好友关系的双方使用。成功响应的 `chat_sent.message` 是服务端持久化后的完整消息；其中 `message_key`、时间戳、过期时间和 `sender_nickname` 由服务端生成或补全。

历史请求的 `limit` 小于等于 0 时使用默认值 25，最大值为 100。服务端按 `created_at` 升序返回一页消息。若仍有更早记录，客户端将本页第一条消息的 `message_key` 放入下一次请求的 `before_message_key`，服务端返回更早的一页；客户端应将新页插入当前列表头部。世界频道最多保留 100 条、私聊每个会话最多保留 50 条，二者最长可读时间均为 24 小时；MongoDB TTL 索引负责过期清理，频道复合索引负责分页查询。

服务端主动推送统一使用 `request_id=0`：

| 推送 | 触发条件 |
| --- | --- |
| `chat_message_pushed` | 世界频道新消息或目标玩家收到私聊消息 |
| `friend_request_received` | 收到好友申请；包含申请人的 `player_id` 和 `nickname` |
| `friend_presence_changed`、`friend_removed` | 好友在线状态或关系变化 |
| `connection_replaced`、`match_result` | 连接被替换或匹配/恢复结果变化 |

世界聊天使用全局 `broadcast` 实时路由，接收该路由的每个 logic-server 会向本机连接广播，并排除已通过 `chat_sent` 收到响应的发送者。私聊使用目标玩家所在实例的 `server` 路由，因此不会向无关玩家泄露消息。完整字段定义以 [`proto/realtime/v1/realtime.proto`](../proto/realtime/v1/realtime.proto) 为准。

## UDP 战斗协议

UDP payload 使用 `proto/battle/v1/session.proto` 的 `ClientPacket` 与 `ServerPacket`。

1. 收到 TCP `matched` 后，向 `battle_udp_addr` 发送 `ClientHello(room_name, player_id, token)`。
2. 收到 `ServerHello` 后保存服务器分配的 conversation，开始接收快照。
3. 每 5 秒发送 `ClientHeartbeat`，并按输入状态发送 `ClientInput`。
4. 在 `BATTLE_PHASE_REWARD_SELECTION` 时，使用 snapshot 中的 `current_options` 发送 `ChooseBlessing`。
5. 在奖励房的 `ROOM_FLOW_STATE_REWARDING` 阶段，每名玩家使用 `ChooseFreeReward` 完成一次免费奖励选择；可选回血、攻击、护甲、随机祝福或 `SKIP`。所有玩家完成后进入 `ROOM_FLOW_STATE_CHOOSING_EXIT`。
6. 奖励房处于 `REWARDING` 或 `CHOOSING_EXIT` 时可发送 `PurchaseShopItem`。商品库存不在玩家间共享，但同一玩家对同一 `item_id` 只能购买一次。

`WorldSnapshot.entities` 中的每个玩家实体都通过 `EntitySnapshot.hero` 携带初始英雄值。客户端在协议边界解析该字段，并在展示层使用 Fire、Ice、Rock、Nature；同一房间内所有客户端收到相同值，断线重连后也保持不变。怪物和投射物的该字段为空。

`EntitySnapshot.collision_radius` 是服务端权威碰撞半径，`kind` 区分玩家、怪物、投射物、障碍物和陷阱。`scene_object_kind` 对静态场景实体给出具体类型：普通障碍物为 `obstacle`，奖励房水泉和商店分别为 `reward_fountain`、`shop`，陷阱使用 `spikes`、`poison_pool` 或 `swamp`。客户端应直接使用这些字段进行渲染和交互表现，不应从坐标或半径反推实体类型。

`WorldSnapshot.player_souls` 在所有房间和阶段发送当局灵魂。`player_combat_stats` 同步每名玩家的攻击伤害、移速、攻击间隔秒数和护甲；客户端显示每秒攻击次数时可使用 `1 / attack_cooldown_seconds`。`player_blessings[].blessings` 是已持有祝福及等级，`current_options` 仅表示当前祝福选择候选。

奖励房快照在 `REWARDING` 阶段发送 `free_reward_states`，并在 `REWARDING`、`CHOOSING_EXIT` 阶段发送 `shop_offers`、`shop_item_definitions` 和 `purchased_shop_items`。协议枚举 `FREE_REWARD_KIND_DAMAGE_REDUCTION` 当前对应实际护甲 `+20`，保留该名称用于协议兼容。免费回血恢复最大生命的 50%，攻击奖励增加 20 点攻击，祝福奖励随机获得或升级一个祝福。

默认商店装备为：重甲（最大生命 `+120`、护甲 `+20`、移速 `-1.5`，30 灵魂）、飞鞋（护甲 `-5`、移速 `+5`，25 灵魂）、护手（攻击 `+10`、最大生命 `+20`，30 灵魂）。近战怪物掉落 10 灵魂，远程怪物掉落 5 灵魂。

默认超时：有效 UDP 输入或心跳缺失 10 秒时 session 断开；房间内所有 session 都断开 90 秒后，服务器以 `all_players_disconnected` 结束房间。完整字段和枚举以 [session.proto](../proto/battle/v1/session.proto) 为准。
