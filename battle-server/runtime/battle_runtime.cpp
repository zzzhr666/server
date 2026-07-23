#include "battle_runtime.hpp"

#include <utility>
#include <vector>
#include <cstdint>

#include "battle_instance.hpp"
#include "game/game_manager.hpp"
#include "net/packet_codec.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"

battle::BattleRuntime::BattleRuntime(RoomManager& room_manager, SessionManager& session_manager,
                                     SendPacketCallback callback)
    : room_manager_(room_manager), session_manager_(session_manager), send_packet_(std::move(callback)) {}

battle::BattleRuntime::~BattleRuntime() = default;

void battle::BattleRuntime::start_room(const std::string& room_name) {
    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (starting_rooms_.contains(room_name) || instances_.contains(room_name)) {
            return;
        }
        starting_rooms_.insert(room_name);
    }

    auto sessions = session_manager_.sessions_in_room(room_name);

    std::vector<std::int64_t> player_ids;
    player_ids.reserve(sessions.size());
    for (const auto& session : sessions) {
        player_ids.push_back(session->player_id());
    }

    auto instance = std::make_unique<BattleInstance>(BattleInstanceConfig{
        .room_name = room_name,
        .player_ids = player_ids,
    });

    auto game_start_packet = make_game_start(room_name, player_ids);

    {
        std::lock_guard<std::mutex> lock(mutex_);
        starting_rooms_.erase(room_name);
        instances_.emplace(room_name, std::move(instance));
    }

    for (const auto& session : sessions) {
        send_packet_(game_start_packet, session->endpoint());
    }
}
