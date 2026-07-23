#pragma once

#include <memory>
#include <string>

#include <grpcpp/grpcpp.h>
#include "generated/proto/rcenter/v1/rcenter.grpc.pb.h"
#include "platform/config.hpp"


namespace battle {
    class RoomManager;

    struct RegisterBattleNodeResult {
        bool ok;
        std::string message;
    };

    class RCenterClient {
    public:
        explicit RCenterClient(std::shared_ptr<grpc::Channel> channel);

        RegisterBattleNodeResult register_battle_node(const Config& config, const RoomManager& room_manager);

    private:
        std::unique_ptr<rcenter::v1::RCenterService::Stub> stub_;
    };
}
