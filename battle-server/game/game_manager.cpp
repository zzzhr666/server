#include "game_manager.hpp"

#include <utility>
#include "spdlog/spdlog.h"

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
    // 房间表写入与 active_players_ 增加必须在同一锁内，rcenter 的节点注册会并发
    // 读取该容量，用它决定此节点是否还能接收新的匹配房间。
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
    SPDLOG_INFO("room manager created room players={} active_players={}", player_count, active_players_);
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
    // 只在房间确实存在时释放预留容量，确保重复 EndRoom 不会导致容量计数下溢。
    active_players_ -= it->second->player_count();
    rooms_.erase(it);
    SPDLOG_INFO("room manager closed room={} active_players={}", room_name, active_players_);
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
    // 复制 shared_ptr 后再调用 Room，避免持有管理器锁时进入 Room 的准入锁。
    return room->can_join(player_id, token);
}

std::vector<battle::PlayerLoadout> battle::RoomManager::player_loadouts(std::string_view room_name) const {
    std::shared_ptr<Room> room;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        const auto it = rooms_.find(std::string(room_name));
        if (it == rooms_.end()) {
            return {};
        }
        room = it->second;
    }
    return room->player_loadouts();
}

std::size_t battle::RoomManager::active_rooms() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return rooms_.size();
}

std::size_t battle::RoomManager::active_players() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return active_players_;
}

battle::JoinRoomResult battle::RoomManager::join_room(const JoinRoomRequest& request) {
    if (request.room_name.empty() || request.token.empty()) {
        return {
            .status = JoinRoomStatus::InvalidRequest,
            .message = "invalid room name",
            .all_players_joined = false
        };
    }
    std::shared_ptr<Room> room;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        const auto it = rooms_.find(request.room_name);
        if (it == rooms_.end()) {
            return {.status = JoinRoomStatus::RoomNotFound, .message = "room not found",.all_players_joined = false};
        }
        room = it->second;
    }
    // Room 自己序列化 token/白名单/重复加入判定；此处只负责安全地定位房间实例。
    return room->join(request.player_id, request.token);
}
