#pragma once
#include <chrono>
#include <string>

namespace battle {
    /// @brief Config 包含本地 battle-server 的地址、容量与超时配置。
    struct Config {
        std::string node_name;
        std::string control_addr;
        std::string udp_bind_addr;
        std::string udp_addr;
        std::string metrics_addr;
        int max_players;
        int tick_rate;
        std::string rcenter_addr;
        std::chrono::seconds session_idle_timeout_seconds;
        std::chrono::seconds all_players_disconnected_timeout;
        std::string log_level;
        std::string log_mode;
    };

    /// @brief DefaultConfig 返回单个本地战斗节点的开发环境默认配置。
    Config DefaultConfig();
}
