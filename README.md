# 游戏服务端

本仓库是一个本地开发用的多人动作游戏服务端。Go 服务负责局外大厅，C++ `battle-server` 负责局内战斗。客户端通过 HTTP 完成注册登录，通过原生 TCP 操作大厅，并通过 UDP 参与战斗。

当前实现包含账号、好友、在线状态、局外成长、双人匹配、四种初始武器、波次战斗、升级祝福、结算奖励，以及断线重连和无人房间清理。

## 服务与端口

| 服务                    | 对宿主机暴露                   | 职责                                 |
|-----------------------|--------------------------|------------------------------------|
| `nginx`               | HTTP `:8080`、TCP `:8081` | 客户端入口，分别转发 HTTP API 和局外实时连接        |
| `logic-1`、`logic-2`   | 否                        | HTTP 注册登录、TCP 认证、好友、在线状态、局外成长和匹配入口 |
| `state`               | 否                        | Go 侧状态服务，唯一直接访问持久化存储的业务进程          |
| `rcenter`             | 否                        | battle 节点注册、匹配队列、活跃对局与结算           |
| `battle-1`、`battle-2` | UDP `:7001`、`:7002`      | 房间、UDP 会话、战斗 tick、ECS 和快照广播        |
| Redis                 | 否                        | 账号、玩家、会话、好友、在线状态、成长与实时事件           |
| MongoDB               | 否                        | 单节点 replica set；为后续聊天历史持久化预留       |

## 快速开始

运行服务依赖 Docker Desktop 和 Docker Compose。Go、CMake、protobuf/gRPC 工具链只在本地运行测试或重新生成协议时需要。

构建并启动全部服务：

```bash
docker compose up -d --build
```

健康检查：

```bash
curl http://localhost:8080/health
```

查看容器状态和日志：

```bash
docker compose ps
docker compose logs -f nginx logic-1 logic-2
```

停止服务但保留 Redis 和 MongoDB 数据卷：

```bash
docker compose down
```

客户端通过 `http://localhost:8080` 注册或登录，取得 token 后连接 `localhost:8081`，首帧完成认证，其余大厅操作均使用 TCP protobuf。battle-server 分别在 UDP `:7001` 和 `:7002` 上接受客户端包；本地下发地址为 `127.0.0.1`，其他机器访问时应在 `compose.yaml` 中将 `--udp-addr` 改为宿主机可访问地址。

## 常用命令

```bash
# Go 全量测试
go test ./...

# 构建并运行 C++ 测试（需要本地 C++ 工具链）
cmake --build battle-server/cmake-build-release-wsl
ctest --test-dir battle-server/cmake-build-release-wsl --output-on-failure

# 重新生成 Go 与 C++ protobuf 代码
bash scripts/generate_proto.sh

# 清除本项目的 Redis 键和 MongoDB 业务库
bash scripts/reset_storage.sh

# 生成统一的 Go、C++ 和 Markdown HTML 文档
bash scripts/generate_docs.sh
```

## 核心行为

- 匹配成功后，rcenter 为每个玩家保存活跃对局信息；TCP 完成认证和 `match_resume` 都可恢复该信息。
- battle-server 在 UDP `hello` 时允许同一玩家重绑到新的会话与端点。
- 连续 15 秒未收到有效 UDP 心跳或输入的会话会标记为断开；所有玩家均断开后，房间等待 90 秒后以 `all_players_disconnected` 原因结束并释放玩家。
- 正常胜利或失败会结算击杀奖励；断线超时结束只释放玩家，不发放奖励。

详细接口见 [docs/api.md](docs/api.md)，服务和战斗架构见 [docs/architecture.md](docs/architecture.md)，ECS 细节见 [battle-server/ecs/design.md](battle-server/ecs/design.md)。
