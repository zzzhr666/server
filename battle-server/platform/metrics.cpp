#include "metrics.hpp"

#include <prometheus/counter.h>
#include <prometheus/gauge.h>
#include <prometheus/histogram.h>
#include <prometheus/registry.h>

namespace {

const prometheus::Histogram::BucketBoundaries& tick_duration_buckets() {
    static const prometheus::Histogram::BucketBoundaries buckets{
        0.001,
        0.005,
        0.010,
        0.016,
        0.020,
        0.033,
        0.050,
        0.100,
        0.250,
        0.500,
        1.000,
    };
    return buckets;
}

}  // namespace

battle::BattleMetrics::BattleMetrics(prometheus::Registry& registry)
    : active_rooms_(
          prometheus::BuildGauge()
              .Name("game_battle_active_rooms")
              .Help("当前 Battle 进程中的活跃房间数。")
              .Register(registry)
              .Add({})),
      active_sessions_(
          prometheus::BuildGauge()
              .Name("game_battle_active_sessions")
              .Help("当前 Battle 进程中已创建且尚未清理的 UDP 会话数。")
              .Register(registry)
              .Add({})),
      control_requests_(
          prometheus::BuildCounter()
              .Name("game_battle_control_requests_total")
              .Help("Battle 控制面操作的累计次数。")
              .Register(registry)),
      udp_packets_(
          prometheus::BuildCounter()
              .Name("game_battle_udp_packets_total")
              .Help("Battle UDP 数据包收发累计次数。")
              .Register(registry)),
      tick_duration_(
          prometheus::BuildHistogram()
              .Name("game_battle_tick_duration_seconds")
              .Help("Battle Runtime 单次 Tick 执行耗时分布。")
              .Register(registry)
              .Add({}, tick_duration_buckets())),
      tick_overruns_(
          prometheus::BuildCounter()
              .Name("game_battle_tick_overruns_total")
              .Help("Battle Runtime 超过目标 Tick 间隔的累计次数。")
              .Register(registry)
              .Add({})) {}

void battle::BattleMetrics::set_active_rooms(std::size_t count) const noexcept {
    active_rooms_.Set(static_cast<double>(count));
}

void battle::BattleMetrics::set_active_sessions(std::size_t count) const noexcept {
    active_sessions_.Set(static_cast<double>(count));
}

void battle::BattleMetrics::observe_control_request(std::string_view operation, bool success) {
    control_requests_
        .Add({
            {"operation", std::string(operation)},
            {"result", success ? "success" : "rejected"},
        })
        .Increment();
}

void battle::BattleMetrics::observe_udp_packet(std::string_view direction, bool accepted) {
    udp_packets_
        .Add({
            {"direction", std::string(direction)},
            {"result", accepted ? "accepted" : "dropped"},
        })
        .Increment();
}

void battle::BattleMetrics::observe_tick_duration(double seconds) {
    tick_duration_.Observe(seconds);
}

void battle::BattleMetrics::increment_tick_overrun() noexcept {
    tick_overruns_.Increment();
}
