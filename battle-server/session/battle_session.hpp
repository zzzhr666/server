#pragma once

#include <chrono>
#include <mutex>
#include <string>
#include <string_view>

#include "net/udp_endpoint.hpp"

namespace battle {
    enum class BattleSessionState:std::uint8_t {
        Connected = 0,
        Disconnected,
        Closed,
    };

    class BattleSession {
    public:
        BattleSession(std::string room_name, std::int64_t player_id, std::uint32_t conv, UdpEndpoint endpoint);

        [[nodiscard]] std::string_view room_name() const {
            return room_name_;
        }

        [[nodiscard]] std::int64_t player_id() const {
            return player_id_;
        }

        [[nodiscard]] BattleSessionState state() const {
            std::lock_guard<std::mutex> lock(mutex_);
            return state_;
        }

        [[nodiscard]] std::uint32_t conv() const {
            std::lock_guard<std::mutex> lock(mutex_);
            return conv_;
        }

        [[nodiscard]] UdpEndpoint endpoint() const {
            std::lock_guard<std::mutex> lock(mutex_);
            return endpoint_;
        }

        void close() {
            std::lock_guard<std::mutex> lock(mutex_);
            state_ = BattleSessionState::Closed;
        }

        void rebind(std::uint32_t conv, UdpEndpoint endpoint);

        bool touch(const UdpEndpoint& endpoint);

        void mark_disconnected();

        bool mark_disconnected_if_stale(std::chrono::steady_clock::time_point now,
                                        std::chrono::steady_clock::duration idle_timeout);

    private:
        std::string room_name_;
        std::int64_t player_id_;
        BattleSessionState state_;
        std::uint32_t conv_;
        UdpEndpoint endpoint_;
        std::chrono::steady_clock::time_point last_seen_at_;
        mutable std::mutex mutex_;
    };
}
