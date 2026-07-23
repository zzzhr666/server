#pragma once
#include <thread>
#include <atomic>

#include "platform/config.hpp"


namespace battle {
    class RCenterClient;
    class RoomManager;

    class NodeRegistrar {
    public:
        NodeRegistrar(Config config, RCenterClient& client, RoomManager& room_manager);
        void start();

        void stop();

    private:
        std::atomic<bool> running_;
        Config config_;
        RCenterClient& rcenter_client_;

        RoomManager& room_manager_;
        std::thread thread_;
    };
}
