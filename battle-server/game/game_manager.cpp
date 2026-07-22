#include "game_manager.hpp"

#include <utility>

battle::RoomManager::RoomManager() : active_players_(0) {}

battle::CreateRoomResult battle::RoomManager::create_room(CreateRoomRequest request) {
    if (request.room_name.empty() || request.token.empty()) {
        return {
            .status = CreateRoomStatus::InvalidRequest,
            .message = "invalid room name or token"
        };
    }
    if (request.player_ids.empty()) {
        return {
            .status = CreateRoomStatus::InvalidRequest,
            .message = "missing player id list"
        };
    }
    std::lock_guard<std::mutex> lock(mutex_);

    if (const auto it = rooms_.find(request.room_name); it != rooms_.end()) {
        return {
            .status = CreateRoomStatus::AlreadyExists,
            .message = "room already exists"
        };
    }
    const auto player_count = request.player_ids.size();
    auto room_name = request.room_name;
    auto room = std::make_shared<Room>(std::move(request));
    rooms_.emplace(std::move(room_name), std::move(room));
    active_players_ += player_count;
    return {
        .status = CreateRoomStatus::OK,
        .message = "room created"
    };
}

bool battle::RoomManager::close_room(std::string_view room_name) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = rooms_.find(std::string(room_name));
    if (it == rooms_.end()) {
        return false;
    }
    active_players_ -= it->second->player_count();
    rooms_.erase(it);
    return true;
}

bool battle::RoomManager::can_join(std::string_view room_name, std::int64_t player_id, std::string_view token) const {
    std::shared_ptr<Room> room;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        const auto it = rooms_.find(std::string(room_name));
        if (it == rooms_.end()) {
            return false;
        }
        room = it->second;
    }
    return room->can_join(player_id, token);
}

std::size_t battle::RoomManager::active_rooms() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return rooms_.size();
}

std::size_t battle::RoomManager::active_players() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return active_players_;
}

battle::JoinRoomResult battle::RoomManager::join_room(JoinRoomRequest request) {
    if (request.room_name.empty() || request.token.empty()) {
        return {.status = JoinRoomStatus::InvalidRequest, .message = "invalid room name"};
    }
    std::shared_ptr<Room> room;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        const auto it = rooms_.find(request.room_name);
        if (it == rooms_.end()) {
            return {.status = JoinRoomStatus::RoomNotFound, .message = "room not found"};
        }
        room = it->second;
    }
    return room->join(request.player_id, request.token);
}
