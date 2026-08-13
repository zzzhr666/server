#include "control_handler.hpp"

#include <utility>

#include "runtime/battle_runtime.hpp"
#include "spdlog/spdlog.h"

battle::ControlHandler::ControlHandler(RoomManager& room_manager, BattleRuntime& battle_runtime)
    : room_manager_(room_manager), battle_runtime_(battle_runtime) {}

battle::CreateRoomResult battle::ControlHandler::create_room(const CreateRoomRequest& request) const {
    const auto room_name = request.room_name;
    auto result = room_manager_.create_room(CreateRoomRequest{request});
    if (result.status == CreateRoomStatus::OK) {
        SPDLOG_INFO("room created room={}", room_name);
    } else {
        SPDLOG_WARN("room creation rejected room={} reason={}", room_name, result.message);
    }
    return result;
}

battle::JoinRoomResult battle::ControlHandler::join_room(const JoinRoomRequest& request) const {
    auto result = room_manager_.join_room(request);
    if (result.status == JoinRoomStatus::OK || result.status == JoinRoomStatus::AlreadyJoined) {
        SPDLOG_DEBUG("room join room={} player={} status={}", request.room_name, request.player_id,
                     result.message);
    } else {
        SPDLOG_WARN("room join rejected room={} player={} reason={}", request.room_name, request.player_id,
                    result.message);
    }
    return result;
}

battle::EndRoomResult battle::ControlHandler::end_room(const EndRoomRequest& request) const {
    if (request.room_name.empty()) {
        return {
            .status = EndRoomStatus::InvalidRequest,
            .message = "invalid room_name"
        };
    }
    const auto reason = request.reason.empty() ? "manual_end" : request.reason;
    auto result = battle_runtime_.end_room(request.room_name, reason);
    if (result.status == EndRoomStatus::OK) {
        SPDLOG_INFO("room ended room={} reason={}", request.room_name, reason);
    } else {
        SPDLOG_WARN("room end rejected room={} reason={}", request.room_name, result.message);
    }
    return result;
}
