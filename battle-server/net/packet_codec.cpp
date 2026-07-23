#include "packet_codec.hpp"

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
                                                std::string reason) {
    v1::ServerPacket packet;
    packet.mutable_game_over()->set_room_name(std::move(room_name));
    for (const auto player_id : player_ids) {
        packet.mutable_game_over()->add_player_ids(player_id);
    }
    packet.mutable_game_over()->set_reason(std::move(reason));
    return packet;
}
