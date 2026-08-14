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
        explicit BattleMetrics(prometheus::Registry& registry);

        void set_active_rooms(std::size_t count) const noexcept;
        void set_active_sessions(std::size_t count) const noexcept;

        void observe_control_request(std::string_view operation, bool success);
        void observe_udp_packet(std::string_view direction, bool accepted);
        void observe_tick_duration(double seconds);
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
