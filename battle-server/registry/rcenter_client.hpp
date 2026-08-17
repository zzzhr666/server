#pragma once

#include <chrono>
#include <memory>
#include <string>

#include <grpcpp/grpcpp.h>
#include "generated/proto/rcenter/v1/rcenter.grpc.pb.h"
#include "platform/config.hpp"


namespace battle {
    struct FinishedBattle;
    class RoomManager;

    /// @brief 战斗节点注册 RPC 的归一化结果。
    struct RegisterBattleNodeResult {
        bool ok;
        std::string message;
    };

    /// @brief 战斗结束通知 RPC 的归一化结果。
    struct FinishMatchResult {
        bool ok;
        std::string message;
    };

    /// @brief RCenterClient 封装 battle-server 到 rcenter-server 的控制 RPC。
    class RCenterClient {
    public:
        RCenterClient(std::shared_ptr<grpc::Channel> channel,
                      std::chrono::seconds register_timeout,
                      std::chrono::seconds finish_timeout);

        /// @brief 上报节点地址和实时房间容量，供 rcenter 调度使用。
        [[nodiscard]] RegisterBattleNodeResult register_battle_node(const Config& config, const RoomManager& room_manager) const;

        /// @brief 通知 rcenter 释放活跃对局并根据结算数据发放奖励。
        [[nodiscard]] FinishMatchResult finish_match(const FinishedBattle& finished) const;

    private:
        std::unique_ptr<rcenter::v1::RCenterService::Stub> stub_;
        std::chrono::seconds register_timeout_;
        std::chrono::seconds finish_timeout_;
    };
}
