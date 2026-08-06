#pragma once
#include <string>
#include <arpa/inet.h>
#include <netinet/in.h>

namespace battle {
    /// @brief UDP 对端的 IPv4 地址和端口。
    struct UdpEndpoint {
        sockaddr_in addr{};
        /// @brief 返回可读的 IPv4 地址字符串。
        [[nodiscard]] std::string ip() const;
        /// @brief 返回主机字节序端口号。
        [[nodiscard]] std::uint16_t port() const;

        bool operator==(const UdpEndpoint& other) const {
            return addr.sin_family == other.addr.sin_family &&
                addr.sin_addr.s_addr == other.addr.sin_addr.s_addr &&
                addr.sin_port == other.addr.sin_port;
        }
    };
}


inline std::string battle::UdpEndpoint::ip() const {
    char buffer[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &addr.sin_addr, buffer, sizeof(buffer));
    return buffer;
}

inline std::uint16_t battle::UdpEndpoint::port() const {
    return ntohs(addr.sin_port);
}
