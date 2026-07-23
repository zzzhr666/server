#include "node_registrar.hpp"

#include "rcenter_client.hpp"

battle::NodeRegistrar::NodeRegistrar(Config config, RCenterClient& client, RoomManager& room_manager)
    : running_(false), config_(std::move(config)), rcenter_client_(client), room_manager_(room_manager) {}

void battle::NodeRegistrar::start() {
    if (running_) {
        return;
    }
    running_ = true;
    thread_ = std::thread([this]() {
        while (running_) {
            rcenter_client_.register_battle_node(config_, room_manager_);
            std::cout << "sent heartbeat..." << std::endl;
            std::this_thread::sleep_for(std::chrono::seconds(3));
        }
    });
}

void battle::NodeRegistrar::stop() {
    if (!running_) {
        return;
    }
    running_ = false;
    if (thread_.joinable()) {
        thread_.join();
    }

}
