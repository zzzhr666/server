#pragma once

#include "control_handler.hpp"
#include "proto/battle/v1/battle.grpc.pb.h"

namespace battle {
    /// @brief BattleControlServiceImpl 将 protobuf 控制 RPC 适配为 ControlHandler 调用。
    class BattleControlServiceImpl final : public v1::BattleControlService::Service {
    public:
        /// @brief 使用控制面处理器创建 BattleControl gRPC 服务。
        explicit BattleControlServiceImpl(ControlHandler& handler);

        /// @brief 将 CreateRoom RPC 转换为房间预留请求及 protobuf 响应。
        grpc::Status CreateRoom(grpc::ServerContext* context,
                                const v1::CreateRoomRequest* request,
                                v1::CreateRoomResponse* response) override;

        /// @brief 将 JoinRoom RPC 转换为房间准入请求及 protobuf 响应。
        grpc::Status JoinRoom(grpc::ServerContext* context,
                              const v1::JoinRoomRequest* request,
                              v1::JoinRoomResponse* response) override;

        /// @brief 将 EndRoom RPC 转换为战斗结束请求及 protobuf 响应。
        grpc::Status EndRoom(grpc::ServerContext* context,
                             const v1::EndRoomRequest* request,
                             v1::EndRoomResponse* response) override;

    private:
        /// @brief 执行控制面领域操作的处理器。
        ControlHandler& control_handler_;
    };
}
