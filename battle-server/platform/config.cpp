#include "config.hpp"

battle::Config battle::DefaultConfig() {
    return {
        .node_name = "battle-demo",
        .control_addr = "127.0.0.1:9101",
        .kcp_addr = "127.0.0.1:7001",
        .max_players = 100,
        .tick_rate = 30,
        .rcenter_addr = "127.0.0.1:9002",
    };
}
