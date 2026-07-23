#pragma once

#include <cstdint>
#include <string>
#include <string_view>

#include "net/udp_endpoint.hpp"

namespace battle {
    enum class BattleSessionState:std::uint8_t {
        Connected = 0,
        Closed = 1,
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
            return state_;
        }

        [[nodiscard]] std::uint32_t conv() const {
            return conv_;
        }

        [[nodiscard]] const UdpEndpoint& endpoint() const {
            return endpoint_;
        }

        void close() {
            state_ = BattleSessionState::Closed;
        }

    private:
        std::string room_name_;
        std::int64_t player_id_;
        BattleSessionState state_;
        std::uint32_t conv_;
        UdpEndpoint endpoint_;
    };
}
