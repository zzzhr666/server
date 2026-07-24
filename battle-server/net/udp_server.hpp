#pragma once

#include <netinet/in.h>

#include <atomic>
#include <cstdint>
#include <string>
#include <string_view>
#include <thread>

#include "proto/battle/v1/session.pb.h"
#include "runtime/battle_runtime.hpp"

namespace battle {
    class SessionManager;

    class UdpServer {
    public:
        UdpServer(std::string listen_addr, SessionManager& session_manager);

        void set_runtime(BattleRuntime& battle_runtime);

        void send_packet(const v1::ServerPacket& packet, const UdpEndpoint& endpoint);

        bool start();

        void stop();

    private:
        void run_loop_();

        std::uint32_t get_next_conv_() {
            return next_conv_.fetch_add(1);
        }

        void send_packet_(const v1::ServerPacket& packet, const sockaddr_in& remote_addr, socklen_t remote_addr_len);

        bool parse_listen_addr_(sockaddr_in& out) const;

        void handle_hello_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr, socklen_t remote_addr_len);

        void handle_move_input(const v1::ClientPacket& packet, const sockaddr_in& remote_addr,
                               socklen_t remote_addr_len);

    private:
        std::string listen_addr_;
        SessionManager& session_manager_;
        BattleRuntime* battle_runtime_;
        std::atomic<bool> running_;
        int fd_;
        std::atomic<std::uint32_t> next_conv_;
        std::thread thread_;
    };
}
