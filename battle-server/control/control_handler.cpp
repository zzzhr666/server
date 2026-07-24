#include "control_handler.hpp"

#include <utility>

#include "runtime/battle_runtime.hpp"

battle::ControlHandler::ControlHandler(RoomManager& room_manager, BattleRuntime& battle_runtime)
    : room_manager_(room_manager), battle_runtime_(battle_runtime) {}

battle::CreateRoomResult battle::ControlHandler::create_room(CreateRoomRequest request) {
    return room_manager_.create_room(std::move(request));
}

battle::JoinRoomResult battle::ControlHandler::join_room(JoinRoomRequest request) {
    return room_manager_.join_room(request);
}

battle::EndRoomResult battle::ControlHandler::end_room(EndRoomRequest request) {
    if (request.room_name.empty()) {
        return {
            .status = EndRoomStatus::InvalidRequest,
            .message = "invalid room_name"
        };
    }
    if (request.reason.empty()) {
        request.reason = "manual_end";
    }
    return battle_runtime_.end_room(request.room_name, request.reason);
}
