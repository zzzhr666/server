#pragma once


#include <functional>
#include <mutex>
#include <string>
#include <memory>
#include <unordered_map>
#include <unordered_set>


#include "net/udp_endpoint.hpp"
#include "proto/battle/v1/session.pb.h"

namespace battle {
    class SessionManager;
    class RoomManager;
    class BattleInstance;

    using SendPacketCallback = std::function<void(const v1::ServerPacket&, const UdpEndpoint&)>;

    class BattleRuntime {
    public:
        BattleRuntime(RoomManager& room_manager, SessionManager& session_manager, SendPacketCallback callback);
        ~BattleRuntime();

        void start_room(const std::string& room_name);

    private:
        RoomManager& room_manager_;
        SessionManager& session_manager_;
        SendPacketCallback send_packet_;
        std::mutex mutex_;
        std::unordered_map<std::string, std::unique_ptr<BattleInstance>> instances_;
        std::unordered_set<std::string> starting_rooms_;
    };
}
