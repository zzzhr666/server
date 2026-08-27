#pragma once

#include <optional>
#include <string_view>
#include <string>
#include <vector>

#include "gameplay/monster_kind.hpp"
#include "generated/proto/battle/v1/session.pb.h"

namespace battle {
    struct PacketMonsterKillCount {
        MonsterKind monster_kind {};
        int count = 0;
    };

    struct PacketPlayerBattleStats {
        std::int64_t player_id = 0;
        int total_kills = 0;
        std::vector<PacketMonsterKillCount> kills;
    };

    /// @brief 从 UDP 负载解码客户端协议包，格式非法时返回空值。
    std::optional<v1::ClientPacket> decode_client_packet(std::string_view bytes);

    /// @brief 将服务端协议包序列化为 UDP 负载。
    std::string encode_server_packet(const v1::ServerPacket& packet);

    /// @brief 创建包含 conversation 与提示信息的握手响应包。
    v1::ServerPacket make_server_hello(std::uint32_t conv, std::string message);

    /// @brief 创建携带房间玩家名单的战斗开始包。
    v1::ServerPacket make_game_start(std::string room_name, const std::vector<std::int64_t>& player_ids);

    /// @brief 创建携带错误码和错误信息的协议包。
    v1::ServerPacket make_error(std::string code, std::string message);

    v1::ServerPacket make_game_over(std::string room_name, const std::vector<std::int64_t>& player_ids,
                                    std::string reason, const std::vector<PacketPlayerBattleStats>& player_stats);
} // battle
