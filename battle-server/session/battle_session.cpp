#include "battle_session.hpp"

battle::BattleSession::BattleSession(std::string room_name, std::int64_t player_id, std::uint32_t conv,
                                     UdpEndpoint endpoint)
    : room_name_(std::move(room_name)), player_id_(player_id), state_(BattleSessionState::Connected),
      conv_(conv), endpoint_(endpoint), last_seen_at_(std::chrono::steady_clock::now()) {}

void battle::BattleSession::rebind(std::uint32_t conv, UdpEndpoint endpoint) {
    std::lock_guard<std::mutex> lock(mutex_);
    // 重连必须同时更新 conversation、源端点和活跃时间。任一字段保留旧值都会
    // 使后续输入被误判为旧会话，或立刻再次因空闲超时断开。
    conv_ = conv;
    endpoint_ = endpoint;
    state_ = BattleSessionState::Connected;
    last_seen_at_ = std::chrono::steady_clock::now();
}

bool battle::BattleSession::touch(const UdpEndpoint& endpoint) {
    std::lock_guard<std::mutex> lock(mutex_);
    // 只允许当前绑定端点刷新 Connected 会话。Disconnected 会话必须通过 hello
    // 走 rebind，避免旧数据包在断线后静默复活会话。
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
    // 仅 Connected 会话能发生一次断线转换；重复扫描不会重复计数，且 Closed
    // 会话由房间清理统一处理。
    if (state_ != BattleSessionState::Connected || now - last_seen_at_ < idle_timeout) {
        return false;
    }
    state_ = BattleSessionState::Disconnected;
    return true;
}
