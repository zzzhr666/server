#include "platform/metrics.hpp"

#include <gtest/gtest.h>

#include <prometheus/registry.h>

#include <string_view>

namespace {

const prometheus::MetricFamily* find_family(const std::vector<prometheus::MetricFamily>& families,
                                            std::string_view name) {
    for (const auto& family : families) {
        if (family.name == name) {
            return &family;
        }
    }
    return nullptr;
}

bool has_label(const prometheus::ClientMetric& metric, std::string_view name, std::string_view value) {
    for (const auto& label : metric.label) {
        if (label.name == name && label.value == value) {
            return true;
        }
    }
    return false;
}

const prometheus::ClientMetric* find_labeled_metric(const prometheus::MetricFamily& family,
                                                    std::string_view first_name,
                                                    std::string_view first_value,
                                                    std::string_view second_name,
                                                    std::string_view second_value) {
    for (const auto& metric : family.metric) {
        if (has_label(metric, first_name, first_value) && has_label(metric, second_name, second_value)) {
            return &metric;
        }
    }
    return nullptr;
}

}  // namespace

TEST(BattleMetricsTest, CollectsGaugesCountersAndTickHistogram) {
    prometheus::Registry registry;
    battle::BattleMetrics metrics(registry);

    metrics.set_active_rooms(2);
    metrics.set_active_sessions(3);
    metrics.observe_control_request("create_room", true);
    metrics.observe_control_request("create_room", false);
    metrics.observe_udp_packet("received", true);
    metrics.observe_udp_packet("received", false);
    metrics.observe_tick_duration(0.012);
    metrics.increment_tick_overrun();

    const auto families = registry.Collect();

    const auto* rooms = find_family(families, "game_battle_active_rooms");
    ASSERT_NE(rooms, nullptr);
    ASSERT_EQ(rooms->metric.size(), 1);
    EXPECT_DOUBLE_EQ(rooms->metric.front().gauge.value, 2.0);

    const auto* sessions = find_family(families, "game_battle_active_sessions");
    ASSERT_NE(sessions, nullptr);
    ASSERT_EQ(sessions->metric.size(), 1);
    EXPECT_DOUBLE_EQ(sessions->metric.front().gauge.value, 3.0);

    const auto* control = find_family(families, "game_battle_control_requests_total");
    ASSERT_NE(control, nullptr);
    const auto* control_success = find_labeled_metric(*control, "operation", "create_room", "result", "success");
    ASSERT_NE(control_success, nullptr);
    EXPECT_DOUBLE_EQ(control_success->counter.value, 1.0);
    const auto* control_rejected = find_labeled_metric(*control, "operation", "create_room", "result", "rejected");
    ASSERT_NE(control_rejected, nullptr);
    EXPECT_DOUBLE_EQ(control_rejected->counter.value, 1.0);

    const auto* udp = find_family(families, "game_battle_udp_packets_total");
    ASSERT_NE(udp, nullptr);
    const auto* udp_accepted = find_labeled_metric(*udp, "direction", "received", "result", "accepted");
    ASSERT_NE(udp_accepted, nullptr);
    EXPECT_DOUBLE_EQ(udp_accepted->counter.value, 1.0);
    const auto* udp_dropped = find_labeled_metric(*udp, "direction", "received", "result", "dropped");
    ASSERT_NE(udp_dropped, nullptr);
    EXPECT_DOUBLE_EQ(udp_dropped->counter.value, 1.0);

    const auto* duration = find_family(families, "game_battle_tick_duration_seconds");
    ASSERT_NE(duration, nullptr);
    ASSERT_EQ(duration->metric.size(), 1);
    EXPECT_EQ(duration->metric.front().histogram.sample_count, 1);
    EXPECT_DOUBLE_EQ(duration->metric.front().histogram.sample_sum, 0.012);

    const auto* overruns = find_family(families, "game_battle_tick_overruns_total");
    ASSERT_NE(overruns, nullptr);
    ASSERT_EQ(overruns->metric.size(), 1);
    EXPECT_DOUBLE_EQ(overruns->metric.front().counter.value, 1.0);
}
