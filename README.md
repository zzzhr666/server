# 游戏服务端

本仓库是一个本地开发用的多人动作游戏服务端。Go 服务负责局外大厅，C++ `battle-server` 负责局内战斗。客户端通过 HTTP/JSON、WebSocket 和 UDP 与服务端交互。

当前实现包含账号、好友、在线状态、局外成长、双人匹配、四种初始武器、波次战斗、升级祝福、结算奖励，以及断线重连和无人房间清理。

## 服务与端口

| 进程 | 默认地址 | 职责 |
| --- | --- | --- |
| `logic-server` | HTTP/WS `:8081`、`:8082` | 客户端 API、认证、好友、在线状态、局外成长、匹配入口 |
| `state-server` | gRPC `127.0.0.1:9001` | Go 侧状态服务，唯一直接访问 Redis 的业务进程 |
| `rcenter-server` | gRPC `127.0.0.1:9002` | battle 节点注册、匹配队列、活跃对局与结算 |
| `battle-server` | `battle-1`: gRPC `127.0.0.1:9101`、UDP `:7001`; `battle-2`: gRPC `127.0.0.1:9102`、UDP `:7002` | 房间、UDP 会话、战斗 tick、ECS 和快照广播 |
| `nginx` | HTTP/WS `:8080` | 可选的本地代理，转发至两个 logic 实例 |
| Redis | `127.0.0.1:6379` | 账号、玩家、会话、好友、在线状态、成长与实时事件 |

## 快速开始

依赖：Go 1.26.5、Redis、CMake 3.20+、C++ protobuf/gRPC/GTest。nginx 仅在需要 `:8080` 统一入口时需要。

启动 Redis：

```bash
redis-server
```

首次配置 C++ Release 构建目录：

```bash
cmake -S battle-server -B battle-server/cmake-build-release-wsl -DCMAKE_BUILD_TYPE=Release
```

启动全部本地服务，不启动 nginx：

```bash
START_NGINX=0 bash scripts/run.sh
```

健康检查：

```bash
curl http://localhost:8081/health
```

默认启用 nginx 时，使用 `http://localhost:8080` 访问 HTTP 与 WebSocket。`scripts/run.sh` 会在后台启动两个 logic-server 实例、state-server、rcenter-server 和两个 battle-server；不带参数时先停止这些服务再启动，`scripts/run.sh start` 只启动，`scripts/run.sh stop` 只停止。服务日志写入 `tmp/logs/`。

两个 battle-server 默认使用 `battle-1`（gRPC `:9101`、UDP `:7001`）和 `battle-2`（gRPC `:9102`、UDP `:7002`）。本地客户端使用默认配置；其他机器访问时，通过 `BATTLE_UDP_PUBLIC_HOST=<可访问IP>` 覆盖下发给客户端的 UDP 主机地址。端口、节点名和容量可用 `BATTLE_1_*`、`BATTLE_2_*` 环境变量覆盖。

## 常用命令

```bash
# Go 全量测试
GOCACHE=/tmp/go-build-cache go test ./...

# 构建并运行 C++ 测试
cmake --build battle-server/cmake-build-release-wsl
ctest --test-dir battle-server/cmake-build-release-wsl --output-on-failure

# 重新生成 Go 与 C++ protobuf 代码
bash scripts/generate_proto.sh

# 删除本项目的 Redis 键
bash scripts/reset_redis.sh
```

WebSocket 和 UDP 手工调试工具：

```bash
go run ./tools/ws_client -url ws://localhost:8081/ws -token <token>
go run ./tools/battle_udp_client -addr 127.0.0.1:7001 -room <room> -token <room-token> -player <player-id>
```

`tools/battle_udp_client` 是调试工具，不是正式客户端实现。

## 核心行为

- 匹配成功后，rcenter 为每个玩家保存活跃对局信息；WebSocket 连接建立和 `match_resume` 都可恢复该信息。
- battle-server 在 UDP `hello` 时允许同一玩家重绑到新的会话与端点。
- 连续 15 秒未收到有效 UDP 心跳或输入的会话会标记为断开；所有玩家均断开后，房间等待 90 秒后以 `all_players_disconnected` 原因结束并释放玩家。
- 正常胜利或失败会结算击杀奖励；断线超时结束只释放玩家，不发放奖励。

详细接口见 [docs/api.md](docs/api.md)，服务和战斗架构见 [docs/architecture.md](docs/architecture.md)，ECS 细节见 [battle-server/ecs/design.md](battle-server/ecs/design.md)。
