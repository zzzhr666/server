#include "packet_codec.hpp"

#include "gameplay/monster_kind_codec.hpp"

std::optional<battle::v1::ClientPacket> battle::decode_client_packet(std::string_view bytes) {
    v1::ClientPacket packet;
    return packet.ParseFromString(bytes) ? std::make_optional(std::move(packet)) : std::nullopt;
}

std::string battle::encode_server_packet(const v1::ServerPacket& packet) {
    return packet.SerializeAsString();
}

battle::v1::ServerPacket battle::make_server_hello(std::uint32_t conv, std::string message) {
    v1::ServerPacket packet;
    packet.mutable_hello()->set_conv(conv);
    packet.mutable_hello()->set_message(std::move(message));
    return packet;
}

battle::v1::ServerPacket battle::make_game_start(std::string room_name, const std::vector<std::int64_t>& player_ids) {
    v1::ServerPacket packet;
    packet.mutable_game_start()->set_room_name(std::move(room_name));
    for (const auto player_id : player_ids) {
        packet.mutable_game_start()->add_player_ids(player_id);
    }
    return packet;
}

battle::v1::ServerPacket battle::make_error(std::string code, std::string message) {
    v1::ServerPacket packet;
    packet.mutable_error()->set_code(std::move(code));
    packet.mutable_error()->set_message(std::move(message));
    return packet;
}

battle::v1::ServerPacket battle::make_game_over(std::string room_name, const std::vector<std::int64_t>& player_ids,
                                                std::string reason,
                                                const std::vector<PacketPlayerBattleStats>& player_stats) {
    v1::ServerPacket packet;
    auto game_over = packet.mutable_game_over();
    game_over->set_room_name(std::move(room_name));
    for (const auto player_id : player_ids) {
        game_over->add_player_ids(player_id);
    }
    game_over->set_reason(std::move(reason));
    for (const auto& player_stat : player_stats) {
        auto proto_player_stat = game_over->add_player_stats();
        proto_player_stat->set_player_id(player_stat.player_id);
        proto_player_stat->set_total_kills(player_stat.total_kills);
        for (const auto& kill : player_stat.kills) {
            auto proto_kill = proto_player_stat->add_kills();
            proto_kill->set_monster_kind(monster_kind_to_string(kill.monster_kind));
            proto_kill->set_count(kill.count);
        }
    }
    return packet;
}
