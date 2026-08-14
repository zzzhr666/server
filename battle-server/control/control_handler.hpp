#pragma once

#include "game/game_manager.hpp"

namespace battle {
    class BattleMetrics;
    class BattleRuntime;
    /// @brief ControlHandler 是外部控制协议调用战斗领域逻辑的应用边界。
    class ControlHandler {
    public:
        ControlHandler(RoomManager& room_manager, BattleRuntime& battle_runtime, BattleMetrics& metrics);

        /// @brief 处理控制面预留房间请求。
        [[nodiscard]] CreateRoomResult create_room(const CreateRoomRequest& request) const;

        /// @brief 处理控制面标记玩家已加入的请求。
        [[nodiscard]] JoinRoomResult join_room(const JoinRoomRequest& request) const;

        /// @brief 处理控制面主动结束房间的请求。
        [[nodiscard]] EndRoomResult end_room(const EndRoomRequest& request) const;

    private:
        /// @brief 房间预留和准入的领域服务。
        RoomManager& room_manager_;
        /// @brief 负责实际战斗生命周期的运行时。
        BattleRuntime& battle_runtime_;
        BattleMetrics& metrics_;
    };
}
