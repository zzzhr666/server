#pragma once


#include <cstdint>
#include <memory>
#include <string>
#include <mutex>
#include <unordered_map>
#include <vector>

#include "net/udp_endpoint.hpp"

namespace battle {
    class RoomManager;
    class BattleSession;

    struct JoinSessionRequest {
        std::string room_name;
        std::string token;
        std::int64_t player_id;
        std::uint32_t conv;
        UdpEndpoint endpoint;
    };

    enum class JoinSessionStatus {
        OK = 0,
        InvalidRequest,
        RoomNotFound,
        InvalidToken,
        PlayerNotAllowed,
        AlreadyJoined,
        InternalError,
    };

    struct JoinSessionResult {
        JoinSessionStatus status;
        std::string message;
        bool all_players_joined;
        std::shared_ptr<BattleSession> session;
    };

    class SessionManager {
    public:
        explicit SessionManager(RoomManager& room_manager);

        JoinSessionResult join(JoinSessionRequest request);

        std::vector<std::shared_ptr<BattleSession>> sessions_in_room(std::string_view room_name) const;

        void remove_room(std::string_view room_name);
    private:
        RoomManager& room_manager_;
        mutable std::mutex mutex_;
        std::unordered_map<std::int64_t,std::shared_ptr<BattleSession>>sessions_by_player_;
        std::unordered_map<std::uint32_t,std::shared_ptr<BattleSession>>sessions_by_conv_;
        std::unordered_map<std::string,std::vector<std::shared_ptr<BattleSession>>>sessions_by_room_;
    };
}
