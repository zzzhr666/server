#include "room_flow.hpp"

#include <algorithm>

battle::RoomFlow::RoomFlow(const DungeonRoomID room_id, const RoomFlowState state) noexcept
    : current_room_id_(room_id), state_(state) {}

bool battle::RoomFlow::transition_to(const RoomFlowState new_state) noexcept {
    switch (new_state) {
    case RoomFlowState::EnteringRoom: {
        if (state_ != RoomFlowState::Transitioning) {
            return false;
        }
        break;
    }
    case RoomFlowState::Fighting: {
        if (state_ != RoomFlowState::EnteringRoom) {
            return false;
        }
        break;
    }
    case RoomFlowState::RoomCleared: {
        if (state_ != RoomFlowState::Fighting) {
            return false;
        }
        break;
    }
    case RoomFlowState::ChoosingBlessing: {
        if (state_ != RoomFlowState::RoomCleared) {
            return false;
        }
        break;
    }
    case RoomFlowState::ChoosingExit: {
        if (state_ != RoomFlowState::RoomCleared && state_ != RoomFlowState::ChoosingBlessing &&
            state_ != RoomFlowState::Rewarding) {
            return false;
        }
        break;
    }
    case RoomFlowState::Transitioning: {
        // 只能由 select_exit() 在验证出口后进入切换阶段。
        return false;
    }
    case RoomFlowState::Rewarding:
        if (state_ != RoomFlowState::EnteringRoom) {
            return false;
        }
        break;
    }
    state_ = new_state;
    return true;
}

bool battle::RoomFlow::select_exit(DungeonRoomID next_room_id, const DungeonRoomGraph& graph) noexcept {
    if (state_ != RoomFlowState::ChoosingExit) {
        return false;
    }
    const auto* room = graph.find_room(current_room_id_);
    if (room == nullptr) {
        return false;
    }
    const auto it = std::ranges::find(room->next_room_ids, next_room_id);
    if (it == room->next_room_ids.end()) {
        return false;
    }
    selected_room_id_ = next_room_id;
    state_ = RoomFlowState::Transitioning;
    return true;
}

bool battle::RoomFlow::complete_transition() noexcept {
    if (state_ != RoomFlowState::Transitioning) {
        return false;
    }
    if (!selected_room_id_.has_value()) {
        return false;
    }
    current_room_id_ = selected_room_id_.value();
    selected_room_id_.reset();
    state_ = RoomFlowState::EnteringRoom;
    return true;
}
