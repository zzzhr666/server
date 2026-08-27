#pragma once

#include <chrono>
#include <mutex>
#include <string>
#include <string_view>

#include "net/udp_endpoint.hpp"

namespace battle {
    /// @brief UDP 战斗会话的连接生命周期状态。
    enum class BattleSessionState:std::uint8_t {
        Connected = 0,
        Disconnected,
        Closed,
    };

    /// @brief BattleSession 将玩家、房间、UDP conversation 和端点绑定为可重连的会话。
    class BattleSession {
    public:
        /// @brief 为房间玩家创建绑定 conversation 与 UDP 端点的会话。
        BattleSession(std::string room_name, std::int64_t player_id, std::uint32_t conv, UdpEndpoint endpoint);

        /// @brief 返回会话所属的房间名。
        [[nodiscard]] std::string_view room_name() const {
            return room_name_;
        }

        /// @brief 返回会话所属的玩家 ID。
        [[nodiscard]] std::int64_t player_id() const {
            return player_id_;
        }

        /// @brief 在线程安全的前提下读取当前连接状态。
        [[nodiscard]] BattleSessionState state() const {
            std::lock_guard<std::mutex> lock(mutex_);
            return state_;
        }

        /// @brief 返回当前 UDP conversation 标识。
        [[nodiscard]] std::uint32_t conv() const {
            std::lock_guard<std::mutex> lock(mutex_);
            return conv_;
        }

        /// @brief 返回当前绑定的 UDP 端点。
        [[nodiscard]] UdpEndpoint endpoint() const {
            std::lock_guard<std::mutex> lock(mutex_);
            return endpoint_;
        }

        /// @brief 永久关闭会话，后续不再允许重连。
        void close() {
            std::lock_guard<std::mutex> lock(mutex_);
            state_ = BattleSessionState::Closed;
        }

        /// @brief 使用新的 conversation 和端点恢复同一玩家会话。
        void rebind(std::uint32_t conv, UdpEndpoint endpoint);

        /// @brief 在端点匹配时刷新最后活跃时间。
        bool touch(const UdpEndpoint& endpoint);

        /// @brief 将尚未关闭的会话标记为断开。
        void mark_disconnected();

        /// @brief 在会话闲置超时后标记断开，并返回是否发生状态变化。
        bool mark_disconnected_if_stale(std::chrono::steady_clock::time_point now,
                                        std::chrono::steady_clock::duration idle_timeout);

    private:
        std::string room_name_;
        std::int64_t player_id_;
        BattleSessionState state_;
        std::uint32_t conv_;
        UdpEndpoint endpoint_;
        /// @brief 最近一次有效输入或心跳的单调时间戳。
        std::chrono::steady_clock::time_point last_seen_at_;
        mutable std::mutex mutex_;
    };
}
