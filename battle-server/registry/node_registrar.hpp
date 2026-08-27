#pragma once
#include <thread>
#include <atomic>

#include "platform/config.hpp"


namespace battle {
    class RCenterClient;
    class RoomManager;

    /// @brief NodeRegistrar 定时向 rcenter 刷新战斗节点注册和容量信息。
    class NodeRegistrar {
    public:
        /// @brief 使用节点配置、rcenter 客户端和房间管理器创建注册器。
        NodeRegistrar(Config config, RCenterClient& client, RoomManager& room_manager);
        /// @brief 启动后台注册刷新线程。
        void start();

        /// @brief 停止后台注册刷新线程。
        void stop();

    private:
        std::atomic<bool> running_;
        Config config_;
        RCenterClient& rcenter_client_;

        RoomManager& room_manager_;
        std::thread thread_;
    };
}
