#pragma once

#include <cstddef>
#include <string_view>

namespace prometheus {
    class Counter;
    class Histogram;
    class Registry;
    class Gauge;
    template <typename T>
    class Family;
}

namespace battle {
    class BattleMetrics {
    public:
        /// @brief 在指定注册表中创建 battle-server 指标。
        explicit BattleMetrics(prometheus::Registry& registry);

        /// @brief 更新当前活跃房间数量。
        void set_active_rooms(std::size_t count) const noexcept;
        /// @brief 更新当前活跃 UDP 会话数量。
        void set_active_sessions(std::size_t count) const noexcept;

        /// @brief 记录一次控制面请求及其结果。
        void observe_control_request(std::string_view operation, bool success);
        /// @brief 记录一个 UDP 数据包的方向与接收结果。
        void observe_udp_packet(std::string_view direction, bool accepted);
        /// @brief 记录一次战斗 tick 的执行耗时。
        void observe_tick_duration(double seconds);
        /// @brief 增加一次 tick 超时计数。
        void increment_tick_overrun() noexcept;

    private:
        prometheus::Gauge& active_rooms_;
        prometheus::Gauge& active_sessions_;
        prometheus::Family<prometheus::Counter>& control_requests_;
        prometheus::Family<prometheus::Counter>& udp_packets_;
        prometheus::Histogram& tick_duration_;
        prometheus::Counter& tick_overruns_;
    };
}
