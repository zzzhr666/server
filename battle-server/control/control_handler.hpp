#pragma once

#include "game/game_manager.hpp"

namespace battle {
    class BattleRuntime;
    /// ControlHandler is the application boundary used by external control transports.
    class ControlHandler {
    public:
        ControlHandler(RoomManager& room_manager, BattleRuntime& battle_runtime);

        /// Handles a control-plane request to reserve a room.
        [[nodiscard]] CreateRoomResult create_room(const CreateRoomRequest& request) const;

        /// Handles a control-plane request to mark a player as joined.
        [[nodiscard]] JoinRoomResult join_room(const JoinRoomRequest& request) const;

        [[nodiscard]] EndRoomResult end_room(const EndRoomRequest& request) const;

    private:
        RoomManager& room_manager_;
        BattleRuntime& battle_runtime_;
    };
}
