#pragma once

#include <cstdint>
#include <optional>
#include <string_view>
#include <string>
#include <vector>

#include "gameplay/monster_kind.hpp"
#include "generated/proto/battle/v1/session.pb.h"

namespace battle {
    struct PacketMonsterKillCount {
        MonsterKind monster_kind;
        int count = 0;
    };

    struct PacketPlayerBattleStats {
        std::int64_t player_id = 0;
        int total_kills = 0;
        std::vector<PacketMonsterKillCount> kills;
    };

    std::optional<v1::ClientPacket> decode_client_packet(std::string_view bytes);

    std::string encode_server_packet(const v1::ServerPacket& packet);

    v1::ServerPacket make_server_hello(std::uint32_t conv, std::string message);

    v1::ServerPacket make_game_start(std::string room_name, const std::vector<std::int64_t>& player_ids);

    v1::ServerPacket make_error(std::string code, std::string message);

    v1::ServerPacket make_game_over(std::string room_name, const std::vector<std::int64_t>& player_ids,
                                    std::string reason, const std::vector<PacketPlayerBattleStats>& player_stats);
} // battle
