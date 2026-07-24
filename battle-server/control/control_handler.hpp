#pragma once

#include "game/game_manager.hpp"

namespace battle {
    class BattleRuntime;
    /// ControlHandler is the application boundary used by external control transports.
    class ControlHandler {
    public:
        ControlHandler(RoomManager& room_manager, BattleRuntime& battle_runtime);

        /// Handles a control-plane request to reserve a room.
        CreateRoomResult create_room(CreateRoomRequest request);

        /// Handles a control-plane request to mark a player as joined.
        JoinRoomResult join_room(JoinRoomRequest request);

        EndRoomResult end_room(EndRoomRequest request);

    private:
        RoomManager& room_manager_;
        BattleRuntime& battle_runtime_;
    };
}
