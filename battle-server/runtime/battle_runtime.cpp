#include "battle_runtime.hpp"

#include <utility>
#include <vector>
#include <cstdint>

#include "game/game_manager.hpp"
#include "net/packet_codec.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"

battle::BattleRuntime::BattleRuntime(RoomManager& room_manager, SessionManager& session_manager,
                                     SendPacketCallback callback)
    : room_manager_(room_manager), session_manager_(session_manager), send_packet_(std::move(callback)) {}

void battle::BattleRuntime::start_room(const std::string& room_name) {
    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (!started_rooms_.insert(room_name).second) {
            return;
        }
    }
    auto sessions = session_manager_.sessions_in_room(room_name);
    std::vector<std::int64_t> player_ids;
    for (const auto& session : sessions) {
        player_ids.push_back(session->player_id());
    }
    auto game_start_packet = make_game_start(room_name, player_ids);
    for (const auto& session : sessions) {
        send_packet_(game_start_packet, session->endpoint());
    }

    //todo: in-game

    auto game_over_packet = make_game_over(room_name, player_ids, "demo_complete");
    for (const auto& session : sessions) {
        send_packet_(game_over_packet, session->endpoint());
    }
    session_manager_.remove_room(room_name);
    room_manager_.close_room(room_name);
    std::lock_guard<std::mutex> lock(mutex_);
    started_rooms_.erase(room_name);
}
