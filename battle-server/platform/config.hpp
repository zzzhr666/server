#pragma once
#include <chrono>
#include <string>

namespace battle {
    /// Config contains local battle-server addresses and capacity settings.
    struct Config {
        std::string node_name;
        std::string control_addr;
        std::string kcp_bind_addr;
        std::string kcp_addr;
        int max_players;
        int tick_rate;
        std::string rcenter_addr;
        std::chrono::seconds session_idle_timeout_seconds;
        std::chrono::seconds all_players_disconnected_timeout;
    };

    /// DefaultConfig returns development defaults for a single local battle node.
    Config DefaultConfig();
}
