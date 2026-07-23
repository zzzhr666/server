#pragma once
#include <cstdint>
#include <string>
#include <arpa/inet.h>
#include <netinet/in.h>

namespace battle {
    struct UdpEndpoint {
        sockaddr_in addr{};
        std::string ip()const;
        std::uint16_t port()const;
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
