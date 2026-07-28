#pragma once

#include <memory>
#include <string>
#include <cstdint>
#include <vector>

#include <grpcpp/grpcpp.h>
#include "generated/proto/rcenter/v1/rcenter.grpc.pb.h"
#include "platform/config.hpp"


namespace battle {
    class RoomManager;

    struct RegisterBattleNodeResult {
        bool ok;
        std::string message;
    };

    struct FinishMatchResult {
        bool ok;
        std::string message;
    };

    class RCenterClient {
    public:
        explicit RCenterClient(std::shared_ptr<grpc::Channel> channel);

        RegisterBattleNodeResult register_battle_node(const Config& config, const RoomManager& room_manager);

        FinishMatchResult finish_match(const std::vector<std::int64_t>& player_ids);

    private:
        std::unique_ptr<rcenter::v1::RCenterService::Stub> stub_;
    };
}
