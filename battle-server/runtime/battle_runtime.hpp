#pragma once


#include <functional>
#include <mutex>
#include <string>
#include <memory>
#include <unordered_map>
#include <unordered_set>
#include <atomic>
#include <thread>


#include "ecs/time.hpp"
#include "game/room.hpp"
#include "net/udp_endpoint.hpp"
#include "proto/battle/v1/session.pb.h"
#include "player_input.hpp"
#include "battle_instance.hpp"

namespace battle {

    class SessionManager;
    class RoomManager;
    class BattleInstance;

    using SendPacketCallback = std::function<void(const v1::ServerPacket&, const UdpEndpoint&)>;


    class BattleRuntime {
    public:
        using BattleInstanceFactory = std::function<std::unique_ptr<BattleInstance>(BattleInstanceConfig)>;
        BattleRuntime(RoomManager& room_manager, SessionManager& session_manager, SendPacketCallback callback,BattleInstanceFactory factory ={});
        ~BattleRuntime();

        void start_room(const std::string& room_name);

        void tick(ecs::DeltaTime delta_time);

        bool receive_input(const std::string& room_name, std::int64_t player_id, PlayerInput input);

        void start();

        void stop();

        EndRoomResult end_room(const std::string& room_name, const std::string& reason);

    private:
        RoomManager& room_manager_;
        SessionManager& session_manager_;
        SendPacketCallback send_packet_;
        std::mutex mutex_;
        std::unordered_map<std::string, std::unique_ptr<BattleInstance>> instances_;
        std::unordered_set<std::string> starting_rooms_;
        std::atomic<bool> running_;
        std::thread tick_thread_;
        BattleInstanceFactory instance_factory_;
    };
}
