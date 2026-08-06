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
            // 周期上报而非仅启动时注册，使 rcenter 看到最新 active_players，
            // 并能在临时网络错误后的下一轮恢复节点可调度状态。
            const auto result = rcenter_client_.register_battle_node(config_, room_manager_);
            if (result.ok) {
                std::cout << "sent heartbeat..." << std::endl;
            } else {
                std::cerr << "register battle node failed: " << result.message << std::endl;
            }
            std::this_thread::sleep_for(std::chrono::seconds(3));
        }
    });
}

void battle::NodeRegistrar::stop() {
    if (!running_) {
        return;
    }
    running_ = false;
    // join 保证后台线程停止访问 config、RoomManager 和 gRPC client 后再析构依赖对象。
    if (thread_.joinable()) {
        thread_.join();
    }

}
