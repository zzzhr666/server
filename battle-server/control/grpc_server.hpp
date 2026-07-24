#pragma once

#include "control_handler.hpp"
#include "proto/battle/v1/battle.grpc.pb.h"

#include <grpcpp/grpcpp.h>

namespace battle {
    /// BattleControlServiceImpl adapts protobuf control RPCs to ControlHandler.
    class BattleControlServiceImpl final : public v1::BattleControlService::Service {
    public:
        explicit BattleControlServiceImpl(ControlHandler& handler);

        grpc::Status CreateRoom(grpc::ServerContext* context,
                                const v1::CreateRoomRequest* request,
                                v1::CreateRoomResponse* response) override;

        grpc::Status JoinRoom(grpc::ServerContext* context,
                              const v1::JoinRoomRequest* request,
                              v1::JoinRoomResponse* response) override;

        grpc::Status EndRoom(grpc::ServerContext* context,
                             const v1::EndRoomRequest* request,
                             v1::EndRoomResponse* response) override;

    private:
        ControlHandler& control_handler_;
    };
}
