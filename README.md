# Game Server Demo

这是一个多进程游戏服务器 demo。当前阶段已经把账号、在线状态、匹配调度和战斗服控制面拆开：

```text
Client
  -> nginx reverse proxy (:8080, optional)
  -> logic-server HTTP/WebSocket (:8081 / :8082)
  -> state-server gRPC (:9001)
  -> Redis

Client
  -> logic-server WebSocket match_start
  -> rcenter-server gRPC (:9002)
  -> battle-server control gRPC (:9101)
  -> battle-server room manager
```

当前 battle-server 先跑通控制面：rcenter 注册 battle 节点后，会缓存该节点的 gRPC control client；匹配成功时 rcenter 直接通知对应 battle-server 创建房间。客户端会从 logic-server 收到 `room_name`、`token` 和 `battle_kcp_addr`，后续再用这些信息连接 battle-server 的实时入口。

## 当前进程

```text
cmd/
├── logic-server/   # HTTP/WebSocket 入口，负责认证、好友、在线状态和匹配入口
├── state-server/   # 状态服务进程，通过 protobuf/gRPC 暴露 Redis 数据操作
└── rcenter-server/ # 资源调度中心，负责 battle 节点注册、匹配队列和创建 battle 房间

battle-server/      # C++ 战斗服，当前提供 BattleControlService gRPC 控制面
```

### logic-server

`logic-server` 对客户端暴露 HTTP API 和 WebSocket 连接。

当前接口：

- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`
- `GET /ws`
- `POST /friends/requests`
- `GET /friends/requests/incoming`
- `GET /friends/requests/outgoing`
- `POST /friends/requests/accept`
- `POST /friends/requests/reject`
- `GET /friends`
- `DELETE /friends`

它不直接操作 Redis，而是通过 `internal/state/grpcclient` 调用 `state-server`。匹配请求从 WebSocket 进入 logic-server，再通过 `internal/rcenter/grpcclient` 调用 `rcenter-server`。

### state-server

`state-server` 是当前唯一直接操作 Redis 的 Go 业务进程。

启动后它会：

1. 连接 Redis。
2. 创建 `internal/state/redisstore.Store`。
3. 创建 `internal/state/service.Service`。
4. 把 state service 注册到 gRPC `StateService`。
5. 监听 `127.0.0.1:9001`。

### rcenter-server

`rcenter-server` 负责资源调度和匹配。

当前能力：

- 通过 `RCenterService.RegisterBattleNode` 注册或刷新 battle 节点。
- 注册时创建并缓存 battle-server control gRPC client。
- `StartMatch` 把玩家放入等待队列，或与最早等待的玩家配对。
- 匹配成功后调用目标 battle-server 的 `BattleControlService.CreateRoom`。
- 返回 `room_name`、`token`、`battle_node_name` 和 `battle_kcp_addr` 给 logic-server。

### battle-server

`battle-server` 是 C++ 战斗服。当前阶段先实现控制面和房间业务：

- `BattleControlService.CreateRoom`
- `BattleControlService.JoinRoom`
- `RoomManager` 管理 active rooms。
- `Room` 校验 room token、允许进入的 player id 和重复 join。

实时战斗入口后续会接 KCP；当前 `kcp_addr` 已经进入 rcenter 的调度结果，但 UDP/KCP 传输层还没有实现。

### nginx

`scripts/run.sh` 可以启动本地 nginx 反向代理，监听 `:8080`，并把请求转发到两个 logic-server 实例：

```text
127.0.0.1:8081 -> logic-1
127.0.0.1:8082 -> logic-2
```

nginx 配置位于 `deploy/nginx/logic.conf`，已经包含 WebSocket upgrade 相关 header。

## 代码结构

```text
internal/
├── battle/
│   └── grpcclient/ # rcenter 使用的 battle control gRPC 客户端适配层
├── contract/
│   ├── battlepb/   # proto/battle/v1/battle.proto 生成的 Go gRPC 代码
│   ├── rcenterpb/  # proto/rcenter/v1/rcenter.proto 生成的 Go gRPC 代码
│   ├── state/      # state-server 对外暴露的业务模型、错误和 Client 接口
│   └── statepb/    # proto/state/v1/state.proto 生成的 Go gRPC 代码
├── logic/
│   ├── auth/       # 注册、登录、登出、会话校验等认证业务
│   ├── friend/     # 好友关系业务
│   ├── match/      # logic 到 rcenter 的匹配业务入口
│   ├── player/     # 玩家资料业务
│   ├── presence/   # 玩家在线状态业务
│   └── httpapi/    # logic-server 的 HTTP/WebSocket 路由和 handler
├── platform/
│   ├── config/     # 进程默认配置
│   └── redisdb/    # Redis client 创建
├── rcenter/
│   ├── grpcclient/ # logic-server 使用的 rcenter gRPC 客户端适配层
│   ├── grpcserver/ # rcenter-server 使用的 gRPC 服务适配层
│   └── rcenterproto/
└── state/
    ├── grpcclient/
    ├── grpcserver/
    ├── redisstore/
    ├── service/
    └── stateproto/

battle-server/
├── control/        # C++ gRPC adapter 和 control handler
├── game/           # C++ Room / RoomManager 业务层
├── generated/      # C++ battle proto 生成物
└── platform/       # C++ battle-server 本地配置
```

几个重要边界：

- `internal/logic/*` 写客户端入口业务，不写 Redis 命令。
- `internal/state/service` 写 state 服务的业务编排，不处理 HTTP 或 WebSocket。
- `internal/state/redisstore` 只负责 Redis 读写和 Redis 侧的一致性控制。
- `internal/rcenter` 写匹配、battle 节点选择和房间创建调度，不直接处理 HTTP。
- `internal/battle/grpcclient` 只适配 battle-server control gRPC，不放 rcenter 业务规则。
- `battle-server/game` 写 C++ 战斗服业务规则，`battle-server/control` 只做控制面适配。

## 配置

默认配置在 `internal/platform/config`：

```text
HTTPAddr:        :8080
StateGRPCAddr:   127.0.0.1:9001
RCenterGRPCAddr: 127.0.0.1:9002
Redis:           127.0.0.1:6379
```

C++ battle-server 当前默认配置在 `battle-server/platform/config.cpp`：

```text
node_name:       battle-demo
control_addr:    127.0.0.1:9101
kcp_bind_addr:   0.0.0.0:7001
kcp_addr:        自动检测到的本机私网 IPv4:7001，例如 WSL 下的 172.x.x.x:7001
max_players:     100
tick_rate:       30
```

`kcp_bind_addr` 是 battle-server 实际监听 UDP 的地址，`kcp_addr` 是注册给 rcenter 并最终发给客户端连接的地址。默认会自动检测非 loopback 私网 IPv4，适合 Unity 跑在 Windows、battle-server 跑在 WSL 的本地开发场景。

如需手动覆盖：

```bash
BATTLE_KCP_BIND_ADDR=0.0.0.0:7001 \
BATTLE_KCP_PUBLIC_ADDR=172.29.93.11:7001 \
./battle-server/cmake-build-debug-wsl/battle_server
```

## 启动

先启动 Redis：

```bash
redis-server
```

启动 C++ battle-server：

```bash
cmake --build battle-server/cmake-build-debug-wsl
./battle-server/cmake-build-debug-wsl/battle_server
```

启动 Go 服务：

```bash
REGISTER_DEMO_BATTLE_NODE=0 bash scripts/run.sh
```

`scripts/run.sh` 会启动 `state-server`、`rcenter-server` 和两个 `logic-server` 实例。battle-server 启动后会自动向 rcenter 注册节点并定时刷新。

如果不想启动 nginx：

```bash
START_NGINX=0 REGISTER_DEMO_BATTLE_NODE=0 bash scripts/run.sh
```

如需绕过 battle-server 自动注册，也可以手动向 rcenter 注册 battle 节点：

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/rcenter/v1/rcenter.proto \
  -d '{"node":{"name":"battle-demo","kcp_addr":"127.0.0.1:7001","control_addr":"127.0.0.1:9101","max_players":100,"active_players":0}}' \
  127.0.0.1:9002 \
  rcenter.v1.RCenterService/RegisterBattleNode
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

WebSocket 在线状态和匹配入口：

```text
GET ws://localhost:8080/ws
Header: token: <token>
```

连接建立后，服务端会把玩家标记为在线。客户端可以发送 heartbeat：

```json
{"type":"heartbeat"}
```

客户端可以发送匹配请求：

```json
{"type":"match_start"}
```

第一个玩家通常收到：

```json
{"type":"match_result","status":"waiting"}
```

第二个玩家匹配成功后会收到：

```json
{
  "type": "match_result",
  "status": "matched",
  "room_name": "room-...",
  "token": "token-...",
  "battle_node_name": "battle-demo",
  "battle_kcp_addr": "127.0.0.1:7001"
}
```

可以直接用 `grpcurl` 验证 battle-server 创建房间：

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/battle/v1/battle.proto \
  -d '{"room_name":"room-e2e","token":"token-e2e","player_ids":[7,8]}' \
  127.0.0.1:9101 \
  battle.v1.BattleControlService/CreateRoom
```

验证 join：

```bash
grpcurl -plaintext \
  -import-path . \
  -proto proto/battle/v1/battle.proto \
  -d '{"room_name":"room-e2e","token":"token-e2e","player_id":7}' \
  127.0.0.1:9101 \
  battle.v1.BattleControlService/JoinRoom
```

查看 Redis 数据：

```bash
redis-cli keys 'game:*'
```

清理本项目 Redis 数据：

```bash
bash scripts/reset_redis.sh
```

## Proto 生成

修改 `proto/state/v1/state.proto`、`proto/rcenter/v1/rcenter.proto` 或 `proto/battle/v1/battle.proto` 后重新生成：

```bash
bash scripts/generate_proto.sh
```

生成结果会写入：

```text
internal/contract/statepb
internal/contract/rcenterpb
internal/contract/battlepb
battle-server/generated
```

## 测试

Go 目标测试：

```bash
GOCACHE=/tmp/go-build-cache go test ./internal/battle/... ./internal/rcenter/... ./cmd/rcenter-server
```

全量 Go 测试：

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

C++ 构建：

```bash
cmake --build battle-server/cmake-build-debug-wsl
```

## 当前限制

- logic/state/rcenter/battle control 已经使用 protobuf/gRPC；生产环境还没有接入 TLS、服务发现或连接池策略。
- C++ battle-server 当前只有控制面和房间业务，还没有实现 UDP/KCP 实时传输。
- `scripts/run.sh` 的 demo battle node flags 与当前 `cmd/rcenter-server` 尚未对齐，本阶段用 `REGISTER_DEMO_BATTLE_NODE=0` 并手动注册 battle 节点。
- `scripts/run.sh` 默认依赖本机 nginx 和 sudo 权限；没有 nginx 时可以用 `START_NGINX=0` 跑两个 logic-server 实例。
- 在线状态目前记录 `online`、`server_name` 和更新时间，并支持 WebSocket heartbeat 续期；好友实时通知已经有基础事件通道，但还不是完整 IM 系统。
