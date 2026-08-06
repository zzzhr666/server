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

    /// @brief UdpServer 负责 UDP 包收发、协议分发和会话入口，不包含玩法规则。
    class UdpServer {
    public:
        UdpServer(std::string listen_addr, SessionManager& session_manager);

        /// @brief 注入处理输入、快照和房间生命周期的运行时。
        void set_runtime(BattleRuntime& battle_runtime);

        /// @brief 将已编码的服务端协议包发送到指定端点。
        void send_packet(const v1::ServerPacket& packet, const UdpEndpoint& endpoint) const;

        /// @brief 绑定监听地址并启动包接收线程。
        bool start();

        /// @brief 停止接收线程并关闭 UDP 套接字。
        void stop();

    private:
        void run_loop_();

        std::uint32_t get_next_conv_() {
            return next_conv_.fetch_add(1);
        }

        void send_packet_(const v1::ServerPacket& packet, const sockaddr_in& remote_addr, socklen_t remote_addr_len) const;

        bool parse_listen_addr_(sockaddr_in& out) const;

        /// @brief 验证 ClientHello，创建或重绑 UDP session 并启动完整房间。
        void handle_hello_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr, socklen_t remote_addr_len);

        /// @brief 验证端点归属后向 BattleRuntime 转交玩家输入。
        void handle_move_input_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr,
                                socklen_t remote_addr_len) const;

        /// @brief 验证端点归属后向 BattleRuntime 转交祝福选择。
        void handle_choose_blessing_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr,
                                     socklen_t remote_addr_len) const;

        /// @brief 刷新匹配会话的活跃时间，避免被断线清理。
        void handle_heartbeat_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr,socklen_t remote_addr_len) const;

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
