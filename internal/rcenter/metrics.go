package rcenter

import "github.com/prometheus/client_golang/prometheus"

// Metrics 保存 rcenter 的业务指标。
type Metrics struct {
	MatchOperations    *prometheus.CounterVec
	MatchQueue         prometheus.Gauge
	ActiveMatchPlayers prometheus.Gauge
	BattleNodes        prometheus.Gauge
}

// NewMetrics 创建并注册 rcenter 业务指标。
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	matchQueuePlayers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_rcenter_match_queue_players",
		Help: "当前处于匹配等待队列中的玩家数量。",
	})
	activeMatchPlayers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_rcenter_active_match_players",
		Help: "当前活跃匹配对局中的玩家数量。",
	})
	battleNodes := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_rcenter_battle_nodes",
		Help: "当前已注册的Battle节点数量。",
	})
	matchOperations := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "game_rcenter_match_operations_total",
		Help: "匹配相关操作的累计次数。",
	}, []string{"operation", "result"})

	registerer.MustRegister(matchQueuePlayers, activeMatchPlayers, battleNodes, matchOperations)
	return &Metrics{
		MatchOperations:    matchOperations,
		MatchQueue:         matchQueuePlayers,
		ActiveMatchPlayers: activeMatchPlayers,
		BattleNodes:        battleNodes,
	}
}
