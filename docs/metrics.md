# 可观测性指标

项目使用 Prometheus 采集 Go 局外服务和 C++ Battle 服务的运行指标。所有服务都使用独立的默认端口 `9200`，在 Docker 网络内由 Prometheus 通过服务名访问；宿主机只需访问 Prometheus 的 `http://localhost:9090`。

## 验证抓取状态

启动服务后执行：

```bash
docker compose up -d --build
curl http://localhost:9090/api/v1/targets
```

`data.activeTargets` 中的 `logic`、`state`、`rcenter` 和 `battle` target 应为 `health: "up"`。直接访问某个容器的指标页面可使用：

```bash
docker compose exec logic-1 wget -qO- http://127.0.0.1:9200/metrics
docker compose exec battle-1 wget -qO- http://127.0.0.1:9200/metrics
```

## Battle 指标

Battle 指标不携带玩家 ID、房间名等高基数标签，只反映进程级负载和协议处理结果。

| 指标 | 类型 | 说明 |
| --- | --- | --- |
| `game_battle_active_rooms` | Gauge | 当前进程持有的活跃房间数 |
| `game_battle_active_sessions` | Gauge | 已创建且尚未清理的 UDP 会话数；断线会话在重连窗口内仍计入 |
| `game_battle_control_requests_total` | Counter | 控制面请求累计次数，标签为 `operation` 和 `result` |
| `game_battle_udp_packets_total` | Counter | UDP 收发包累计次数，标签为 `direction` 和 `result` |
| `game_battle_tick_duration_seconds` | Histogram | 单次 Runtime tick 执行耗时 |
| `game_battle_tick_overruns_total` | Counter | tick 执行耗时超过目标 tick 间隔的次数 |

`result="accepted"` 表示 UDP 包已通过协议层识别，不代表后续业务请求一定成功；业务层失败应结合日志和控制面指标判断。

## 常用 PromQL

Battle 当前房间和会话：

```promql
game_battle_active_rooms
game_battle_active_sessions
```

最近一分钟 UDP 收发速率：

```promql
sum by (job, instance, direction, result) (
  rate(game_battle_udp_packets_total[1m])
)
```

最近五分钟 tick P95 耗时：

```promql
histogram_quantile(
  0.95,
  sum by (job, instance, le) (
    rate(game_battle_tick_duration_seconds_bucket[5m])
  )
)
```

最近五分钟 tick 超时次数：

```promql
increase(game_battle_tick_overruns_total[5m])
```

控制面拒绝率：

```promql
sum by (job, instance, operation) (
  rate(game_battle_control_requests_total{result="rejected"}[5m])
)
```

Go 服务还会暴露 Go runtime、进程资源和各领域业务指标；可以在 Prometheus 查询页面按 `job`、`instance` 和指标名称筛选。日志用于解释已经发生的单次事件，Prometheus 指标用于观察当前负载、趋势和异常率。
