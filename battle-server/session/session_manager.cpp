#include "session_manager.hpp"

#include <ranges>

#include "battle_session.hpp"
#include "game/game_manager.hpp"
#include "game/room.hpp"
#include "spdlog/spdlog.h"
#include "platform/metrics.hpp"

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

battle::SessionManager::SessionManager(RoomManager& room_manager, BattleMetrics& metrics)
    : metrics_(metrics), room_manager_(room_manager) {}

battle::JoinSessionResult battle::SessionManager::join(JoinSessionRequest request) {
    if (request.room_name.empty() || request.token.empty() || request.player_id <= 0 || request.conv == 0) {
        return {
            .status = JoinSessionStatus::InvalidRequest,
            .message = "invalid request",
            .all_players_joined = false,
            .session = nullptr
        };
    }
    // 先在锁内取得共享指针，随后在锁外执行房间/令牌校验，避免 SessionManager
    // 锁与 RoomManager 锁交叉持有。重新加锁时会再次确认对象仍是当前索引值。
    std::shared_ptr<BattleSession> existing_session;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (const auto it = sessions_by_player_.find(request.player_id); it != sessions_by_player_.end()) {
            existing_session = it->second;
        }
    }
    if (existing_session) {
        if (existing_session->room_name() != request.room_name) {
            return {
                .status = JoinSessionStatus::PlayerNotAllowed,
                .message = "invalid request",
                .all_players_joined = false,
                .session = nullptr
            };
        }
        if (!room_manager_.can_join(request.room_name, request.player_id, request.token)) {
            return {
                .status = JoinSessionStatus::InvalidToken,
                .message = "invalid token",
                .all_players_joined = false,
                .session = nullptr,
            };
        }
        // 同一玩家 hello 到达时是重连而非第二次加入。替换 conv 和 endpoint 前，
        // 必须先删除旧 conv 索引，保证一个 conversation 永远只指向一个会话。
        std::lock_guard<std::mutex> lock(mutex_);
        const auto player_it = sessions_by_player_.find(request.player_id);
        if (player_it == sessions_by_player_.end() || player_it->second != existing_session) {
            return {
                .status = JoinSessionStatus::RoomNotFound,
                .message = "session no longer exists",
                .all_players_joined = false,
                .session = nullptr,
            };
        }
        const auto conv_it = sessions_by_conv_.find(request.conv);
        if (conv_it != sessions_by_conv_.end() && conv_it->second != existing_session) {
            return {
                .status = JoinSessionStatus::InternalError,
                .message = "conversation already in use",
                .all_players_joined = false,
                .session = nullptr,
            };
        }
        const auto old_conv = existing_session->conv();
        sessions_by_conv_.erase(old_conv);

        existing_session->rebind(request.conv, request.endpoint);
        sessions_by_conv_[request.conv] = existing_session;
        SPDLOG_DEBUG("session rebound room={} player={} conv={}", request.room_name, request.player_id, request.conv);
        return {
            .status = JoinSessionStatus::AlreadyJoined,
            .message = "session reconnected",
            .all_players_joined = false,
            .session = existing_session,
        };
    }

    // 新玩家必须先获得 Room 的准入许可；只有 Room 成功记录 joined 状态后，
    // 才创建三个会话索引，避免无效 token 留下半成品 session。
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

    // Room 准入与插入 session 索引之间可能有并发 hello。这里再次检查，
    // 让后到请求复用已建立会话，而不是覆盖已有玩家或 conversation。
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
    metrics_.set_active_sessions(sessions_by_player_.size());
    SPDLOG_INFO("session created room={} player={} conv={}", session->room_name(), session->player_id(), session->conv());

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
    // 关闭顺序同时清理 player、conversation 和 room 三个索引，确保房间结束后
    // 旧 UDP 包无法通过 touch 恢复已释放的战斗实例。
    for (const auto& session : it->second) {
        sessions_by_conv_.erase(session->conv());
        sessions_by_player_.erase(session->player_id());
        session->close();
    }
    sessions_by_room_.erase(it);
    metrics_.set_active_sessions(sessions_by_player_.size());
}

bool battle::SessionManager::touch(std::string_view room_name, std::int64_t player_id, const UdpEndpoint& endpoint) {
    std::lock_guard<std::mutex> lock(mutex_);
    const auto it = sessions_by_player_.find(player_id);
    if (it == sessions_by_player_.end()) {
        return false;
    }
    const auto& session = it->second;
    return session->room_name() == room_name ? session->touch(endpoint) : false;
}

std::size_t battle::SessionManager::mark_stale_sessions(std::chrono::steady_clock::time_point now,
                                                        std::chrono::steady_clock::duration idle_timeout) {
    std::vector<std::shared_ptr<BattleSession>> sessions;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        sessions.reserve(sessions_by_player_.size());
        for (const auto& session : sessions_by_player_ | std::views::values) {
            sessions.emplace_back(session);
        }
    }
    // 先复制 shared_ptr 再在锁外切换 session 状态，避免长时间持有管理器锁；
    // session 自身有 mutex，因此重绑和超时判定仍保持线程安全。
    std::size_t disconnected_count = 0;
    for (const auto& session : sessions) {
        if (session->mark_disconnected_if_stale(now,idle_timeout)) {
            ++disconnected_count;
        }
    }
    return disconnected_count;
}

std::vector<std::shared_ptr<battle::BattleSession>> battle::SessionManager::connected_sessions_in_room(
    std::string_view room_name) const {
    auto sessions = sessions_in_room(room_name);
    std::vector<std::shared_ptr<BattleSession>>connected_sessions;
    connected_sessions.reserve(sessions.size());
    for (const auto& session : sessions) {
        if (session->state() == BattleSessionState::Connected) {
            connected_sessions.emplace_back(session);
        }
    }
    return connected_sessions;
}
