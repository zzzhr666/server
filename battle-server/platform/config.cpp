#include "config.hpp"

#include <cstdlib>
#include <ifaddrs.h>
#include <net/if.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <optional>
#include <string>
#include <charconv>

namespace {
    constexpr const char* kDefaultBattleUdpPort = "7001";

    std::optional<std::string> getenv_string(const char* name) {
        const char* value = std::getenv(name);
        if (value == nullptr || value[0] == '\0') {
            return std::nullopt;
        }
        return std::string(value);
    }

    bool is_private_ipv4(const std::string& ip) {
        if (ip.rfind("10.", 0) == 0 || ip.rfind("192.168.", 0) == 0) {
            return true;
        }
        if (ip.rfind("172.", 0) != 0) {
            return false;
        }
        const auto second_start = ip.find('.') + 1;
        const auto second_end = ip.find('.', second_start);
        if (second_start == std::string::npos || second_end == std::string::npos) {
            return false;
        }
        int octet = 0;
        const auto* begin = ip.data() + second_start;
        const auto* end = ip.data() + second_end;
        const auto result = std::from_chars(begin, end, octet);
        if (result.ec != std::errc{} || result.ptr != end) {
            return false;
        }
        return octet >= 16 && octet <= 31;
    }

    std::optional<std::string> detect_private_ipv4() {
        ifaddrs* interfaces = nullptr;
        if (getifaddrs(&interfaces) != 0) {
            return std::nullopt;
        }

        std::optional<std::string> fallback;
        for (auto* iface = interfaces; iface != nullptr; iface = iface->ifa_next) {
            if (iface->ifa_addr == nullptr || iface->ifa_addr->sa_family != AF_INET) {
                continue;
            }
            if ((iface->ifa_flags & IFF_LOOPBACK) != 0) {
                continue;
            }

            char buffer[INET_ADDRSTRLEN] = {};
            auto* addr = reinterpret_cast<sockaddr_in*>(iface->ifa_addr);
            if (inet_ntop(AF_INET, &addr->sin_addr, buffer, sizeof(buffer)) == nullptr) {
                continue;
            }

            std::string ip = buffer;
            if (is_private_ipv4(ip)) {
                freeifaddrs(interfaces);
                return ip;
            }
            if (!fallback.has_value()) {
                fallback = std::move(ip);
            }
        }

        freeifaddrs(interfaces);
        return fallback;
    }

    std::string default_public_kcp_addr() {
        if (auto env = getenv_string("BATTLE_KCP_PUBLIC_ADDR")) {
            return *env;
        }
        auto ip = detect_private_ipv4().value_or("127.0.0.1");
        return ip + ":" + kDefaultBattleUdpPort;
    }

    std::string default_bind_kcp_addr() {
        return getenv_string("BATTLE_KCP_BIND_ADDR").value_or("0.0.0.0:" + std::string(kDefaultBattleUdpPort));
    }
}

battle::Config battle::DefaultConfig() {
    return {
        .node_name = "battle-demo",
        .control_addr = "127.0.0.1:9101",
        .kcp_bind_addr = default_bind_kcp_addr(),
        .kcp_addr = default_public_kcp_addr(),
        .max_players = 100,
        .tick_rate = 30,
        .rcenter_addr = "127.0.0.1:9002",
    };
}
