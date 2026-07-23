#include "room.hpp"

#include <algorithm>
#include <utility>

battle::Room::Room(CreateRoomRequest request)
    : room_name_(std::move(request.room_name)),
      token_(std::move(request.token)),
      allowed_player_ids_(std::move(request.player_ids)) {}

bool battle::Room::can_join(int64_t player_id, std::string_view token) const {
    if (token.empty() || token != token_) {
        return false;
    }
    return std::ranges::any_of(allowed_player_ids_, [player_id](std::int64_t x) {
        return x == player_id;
    });
}

battle::JoinRoomResult battle::Room::join(std::int64_t player_id, std::string_view token) {
    if (player_id <= 0 || token.empty()) {
        return {.status = JoinRoomStatus::InvalidRequest, .message = "invalid request", .all_players_joined = false};
    }

    std::lock_guard<std::mutex> lock(join_mutex_);

    if (token != token_) {
        return {
            .status = JoinRoomStatus::InvalidToken, .message = "token does not match request",
            .all_players_joined = false
        };
    }

    if (!std::ranges::any_of(allowed_player_ids_, [player_id](std::int64_t x) {
        return x == player_id;
    })) {
        return {
            .status = JoinRoomStatus::PlayerNotAllowed, .message = "player not allowed", .all_players_joined = false
        };
    }

    if (joined_player_ids_.contains(player_id)) {
        return {
            .status = JoinRoomStatus::AlreadyJoined, .message = "player already joined", .all_players_joined = false
        };
    }
    joined_player_ids_.insert(player_id);
    return {
        .status = JoinRoomStatus::OK, .message = "player joined",
        .all_players_joined = allowed_player_ids_.size() == joined_player_ids_.size()
    };
}
