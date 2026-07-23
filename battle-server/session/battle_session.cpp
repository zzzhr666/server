#include "battle_session.hpp"

battle::BattleSession::BattleSession(std::string room_name, std::int64_t player_id, std::uint32_t conv,
                                     UdpEndpoint endpoint)
    : room_name_(std::move(room_name)), player_id_(player_id), state_(BattleSessionState::Connected),
      conv_(conv), endpoint_(endpoint) {}
