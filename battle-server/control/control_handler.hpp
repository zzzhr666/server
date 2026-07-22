#pragma once

#include "game/game_manager.hpp"

namespace battle {
    /// ControlHandler is the application boundary used by external control transports.
    class ControlHandler {
    public:
        explicit ControlHandler(RoomManager& room_manager);

        /// Handles a control-plane request to reserve a room.
        CreateRoomResult create_room(CreateRoomRequest request);

        /// Handles a control-plane request to mark a player as joined.
        JoinRoomResult join_room(JoinRoomRequest request);

    private:
        RoomManager& room_manager_;
    };
}
