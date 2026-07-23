#include "session_manager.hpp"

#include "battle_session.hpp"
#include "game/game_manager.hpp"
#include "game/room.hpp"


namespace {
    battle::JoinSessionStatus from_join_room_status(battle::JoinRoomStatus status) {
        switch (status) {
        case battle::JoinRoomStatus::OK:
            return battle::JoinSessionStatus::OK;
        case battle::JoinRoomStatus::AlreadyJoined:
            return battle::JoinSessionStatus::AlreadyJoined;
        case battle::JoinRoomStatus::InvalidToken:
            return battle::JoinSessionStatus::InvalidToken;
        case battle::JoinRoomStatus::PlayerNotAllowed:
            return battle::JoinSessionStatus::PlayerNotAllowed;
        case battle::JoinRoomStatus::RoomNotFound:
            return battle::JoinSessionStatus::RoomNotFound;
        case battle::JoinRoomStatus::InvalidRequest:
            return battle::JoinSessionStatus::InvalidRequest;
        case battle::JoinRoomStatus::InternalError:
            return battle::JoinSessionStatus::InternalError;
        }
        return battle::JoinSessionStatus::InternalError;
    }
}

battle::SessionManager::SessionManager(RoomManager& room_manager)
    : room_manager_(room_manager) {}

battle::JoinSessionResult battle::SessionManager::join(JoinSessionRequest request) {
    if (request.room_name.empty() || request.token.empty() || request.player_id <= 0 || request.conv == 0) {
        return {
            .status = JoinSessionStatus::InvalidRequest,
            .message = "invalid request",
            .all_players_joined = false,
            .session = nullptr
        };
    }

    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (const auto it = sessions_by_player_.find(request.player_id); it != sessions_by_player_.end()) {
            return {
                .status = JoinSessionStatus::AlreadyJoined,
                .message = "existing session",
                .all_players_joined = false,
                .session = it->second
            };
        }
        if (const auto it = sessions_by_conv_.find(request.conv); it != sessions_by_conv_.end()) {
            return {
                .status = JoinSessionStatus::AlreadyJoined,
                .message = "existing session",
                .all_players_joined = false,
                .session = it->second
            };
        }
    }
    JoinRoomResult res = room_manager_.join_room({
        .room_name = request.room_name,
        .token = request.token,
        .player_id = request.player_id
    });
    if (res.status == JoinRoomStatus::AlreadyJoined) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = sessions_by_player_.find(request.player_id);
        if (it == sessions_by_player_.end()) {
            return {
                .status = JoinSessionStatus::InternalError,
                .message = "joined room without session",
                .all_players_joined = false,
                .session = nullptr
            };
        }
        return {
            .status = JoinSessionStatus::AlreadyJoined,
            .message = "already joined",
            .all_players_joined = false,
            .session = it->second
        };
    }
    if (res.status != JoinRoomStatus::OK) {
        return {
            .status = from_join_room_status(res.status),
            .message = res.message,
            .all_players_joined = false,
            .session = nullptr
        };
    }

    auto session = std::make_shared<BattleSession>(std::move(request.room_name), request.player_id, request.conv,
                                                   request.endpoint);

    std::lock_guard<std::mutex> lock(mutex_);
    if (const auto it = sessions_by_player_.find(session->player_id()); it != sessions_by_player_.end()) {
        return {
            .status = JoinSessionStatus::AlreadyJoined,
            .message = "existing session",
            .all_players_joined = false,
            .session = it->second
        };
    }
    if (const auto it = sessions_by_conv_.find(session->conv()); it != sessions_by_conv_.end()) {
        return {
            .status = JoinSessionStatus::AlreadyJoined,
            .message = "existing session",
            .all_players_joined = false,
            .session = it->second
        };
    }
    sessions_by_player_[request.player_id] = session;
    sessions_by_conv_[request.conv] = session;
    sessions_by_room_[std::string(session->room_name())].push_back(session);

    return {
        .status = JoinSessionStatus::OK,
        .message = "join success",
        .all_players_joined = res.all_players_joined,
        .session = session
    };
}

std::vector<std::shared_ptr<battle::BattleSession>>
battle::SessionManager::sessions_in_room(std::string_view room_name) const {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = sessions_by_room_.find(std::string(room_name));
    if (it == sessions_by_room_.end()) {
        return {};
    }
    return it->second;
}

void battle::SessionManager::remove_room(std::string_view room_name) {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = sessions_by_room_.find(std::string(room_name));
    if (it == sessions_by_room_.end()) {
        return;
    }
    for (const auto& session : it->second) {
        sessions_by_conv_.erase(session->conv());
        sessions_by_player_.erase(session->player_id());
        session->close();
    }
    sessions_by_room_.erase(it);
}
