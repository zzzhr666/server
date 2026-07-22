#include "control_handler.hpp"

#include <utility>

battle::ControlHandler::ControlHandler(RoomManager& room_manager) : room_manager_(room_manager) {}

battle::CreateRoomResult battle::ControlHandler::create_room(CreateRoomRequest request) {
    return room_manager_.create_room(std::move(request));
}

battle::JoinRoomResult battle::ControlHandler::join_room(JoinRoomRequest request) {
    return room_manager_.join_room(std::move(request));
}
