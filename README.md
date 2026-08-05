# Game Server Demo

这是一个本地开发用的多人游戏服务器 demo。大厅、账号、好友、在线状态和匹配由 Go 服务负责；战斗房间和局内 ECS 由 C++ `battle-server` 负责。

## 进程

| 进程 | 入口 | 默认地址 | 职责 |
| --- | --- | --- | --- |
| `logic-server` | `cmd/logic-server` | HTTP/WS `:8081`, `:8082` | 客户端 HTTP API、WebSocket、认证、好友、在线状态、匹配入口 |
| `state-server` | `cmd/state-server` | gRPC `127.0.0.1:9001` | 统一读写 Redis 状态 |
| `rcenter-server` | `cmd/rcenter-server` | gRPC `127.0.0.1:9002` | battle 节点注册、匹配队列、创建战斗房间 |
| `battle-server` | `battle-server/main.cpp` | gRPC `127.0.0.1:9101`, UDP `:7001` | 战斗房间、UDP 会话、ECS tick、快照广播 |
| `nginx` | `deploy/nginx/logic.conf` | HTTP/WS `:8080` | 本地可选反向代理，转发到两个 logic 实例 |
| `Redis` | external | `127.0.0.1:6379` | 存储账号、玩家、会话、在线状态、好友、实时事件 |

## Quickstart

依赖：

- Go 1.25+
- Redis
- CMake 3.20+
- C++ protobuf、gRPC、GTest
- nginx 可选，只在需要 `:8080` 代理入口时使用

启动 Redis：

```bash
redis-server
```

首次配置 C++ battle-server：

```bash
cmake -S battle-server -B battle-server/cmake-build-release-wsl
```

启动本地服务，不启用 nginx：

```bash
START_NGINX=0 bash scripts/run.sh
```

健康检查：

```bash
curl http://localhost:8081/health
```

如果本机有 nginx 和 sudo 权限，也可以使用默认代理入口：

```bash
bash scripts/run.sh
curl http://localhost:8080/health
```

## 常用验证

注册：

```bash
curl -i -X POST http://localhost:8081/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123","nickname":"Alice"}'
```

登录：

```bash
curl -i -X POST http://localhost:8081/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}'
```

查询当前玩家：

```bash
curl -i http://localhost:8081/auth/me \
  -H 'Authorization: Bearer <token>'
```

WebSocket 调试：

```bash
go run ./tools/ws_client -url ws://localhost:8081/ws -token <token>
```

连接后可以输入：

```json
{"type":"heartbeat"}
```

```json
{"type":"match_start"}
```

UDP 战斗调试：

```bash
go run ./tools/battle_udp_client \
  -addr 127.0.0.1:7001 \
  -room <room_name> \
  -token <room_token> \
  -player <player_id> \
  -move-x 1 \
  -move-y 0
```

## 常用命令

运行 Go 测试：

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

构建 C++ battle-server：

```bash
cmake --build battle-server/cmake-build-debug-wsl
```

运行 C++ 测试：

```bash
./battle-server/cmake-build-debug-wsl/battle_ecs_tests
./battle-server/cmake-build-debug-wsl/battle_runtime_tests
```

重新生成 proto：

```bash
bash scripts/generate_proto.sh
```

清理本项目 Redis 数据：

```bash
bash scripts/reset_redis.sh
```

手动创建或结束 battle room：

```bash
go run ./tools/create_battle_room -room room-1 -token token-1 -players 1001,1002
go run ./tools/end_battle_room -room room-1 -reason manual_end
```

## 目录

```text
cmd/
├── logic-server/
├── state-server/
└── rcenter-server/

internal/
├── logic/      # HTTP/WS 入口业务：auth、friend、match、player、presence
├── state/      # state gRPC server、Redis store、状态服务
├── rcenter/    # battle 节点注册和匹配调度
├── battle/     # Go 侧 battle control gRPC client
├── contract/   # protobuf 生成代码和共享状态契约
└── platform/   # 本地配置和 Redis client

battle-server/
├── control/    # C++ battle control gRPC
├── ecs/        # 局内 ECS components / systems / world
├── game/       # Room / RoomManager
├── net/        # UDP packet 收发
├── runtime/    # BattleRuntime / BattleInstance
└── session/    # UDP session 管理

proto/
├── battle/v1/
├── rcenter/v1/
└── state/v1/
```

更多接口见 [docs/api.md](docs/api.md)，当前架构见 [docs/architecture.md](docs/architecture.md)。
