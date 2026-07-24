#pragma once
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
    };

    /// DefaultConfig returns development defaults for a single local battle node.
    Config DefaultConfig();
}
