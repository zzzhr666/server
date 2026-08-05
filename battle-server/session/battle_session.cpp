#include "battle_session.hpp"

battle::BattleSession::BattleSession(std::string room_name, std::int64_t player_id, std::uint32_t conv,
                                     UdpEndpoint endpoint)
    : room_name_(std::move(room_name)), player_id_(player_id), state_(BattleSessionState::Connected),
      conv_(conv), endpoint_(endpoint), last_seen_at_(std::chrono::steady_clock::now()) {}

void battle::BattleSession::rebind(std::uint32_t conv, UdpEndpoint endpoint) {
    std::lock_guard<std::mutex> lock(mutex_);
    conv_ = conv;
    endpoint_ = endpoint;
    state_ = BattleSessionState::Connected;
    last_seen_at_ = std::chrono::steady_clock::now();
}

bool battle::BattleSession::touch(const UdpEndpoint& endpoint) {
    std::lock_guard<std::mutex> lock(mutex_);
    if (state_ != BattleSessionState::Connected || endpoint_ != endpoint) {
        return false;
    }
    last_seen_at_ = std::chrono::steady_clock::now();
    return true;
}

void battle::BattleSession::mark_disconnected() {
    std::lock_guard<std::mutex> lock(mutex_);
    state_ = BattleSessionState::Disconnected;
}

bool battle::BattleSession::mark_disconnected_if_stale(std::chrono::steady_clock::time_point now,
    std::chrono::steady_clock::duration idle_timeout) {
    std::lock_guard<std::mutex> lock(mutex_);
    if (state_ != BattleSessionState::Connected || now - last_seen_at_ < idle_timeout) {
        return false;
    }
    state_ = BattleSessionState::Disconnected;
    return true;
}
