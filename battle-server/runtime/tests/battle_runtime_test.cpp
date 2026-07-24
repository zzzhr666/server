#include "runtime/battle_runtime.hpp"

#include <utility>
#include <vector>

#include <gtest/gtest.h>

#include "game/game_manager.hpp"
#include "session/session_manager.hpp"

namespace battle {
namespace {

UdpEndpoint endpoint_with_port(std::uint16_t port) {
    UdpEndpoint endpoint;
    endpoint.addr.sin_family = AF_INET;
    endpoint.addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    endpoint.addr.sin_port = htons(port);
    return endpoint;
}

TEST(BattleRuntimeTest, ReceiveInputAndTickBroadcastsMovedSnapshot) {
    RoomManager room_manager;
    SessionManager session_manager(room_manager);
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    BattleRuntime runtime(room_manager, session_manager,
                          [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
                              sent_packets.emplace_back(packet, endpoint);
                          });

    auto create_result = room_manager.create_room({
        .room_name = "room-1",
        .token = "token-1",
        .player_ids = {1001, 1002},
    });
    ASSERT_EQ(create_result.status, CreateRoomStatus::OK);
    ASSERT_EQ(session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1001,
                  .conv = 1,
                  .endpoint = endpoint_with_port(7001),
              }).status,
              JoinSessionStatus::OK);
    ASSERT_EQ(session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1002,
                  .conv = 2,
                  .endpoint = endpoint_with_port(7002),
              }).status,
              JoinSessionStatus::OK);

    runtime.start_room("room-1");
    sent_packets.clear();

    ASSERT_TRUE(runtime.receive_input("room-1", 1002, 1.0f, 0.0f));
    runtime.tick(ecs::DeltaTime{1.0f});

    ASSERT_EQ(sent_packets.size(), 2);
    for (const auto& [packet, endpoint] : sent_packets) {
        ASSERT_EQ(packet.payload_case(), v1::ServerPacket::kSnapshot);
        EXPECT_EQ(packet.snapshot().room_name(), "room-1");
        ASSERT_EQ(packet.snapshot().entities_size(), 2);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(0).x_position(), -2.0f);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(0).y_position(), 0.0f);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(1).x_position(), 7.0f);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(1).y_position(), 0.0f);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(1).x_direction(), 1.0f);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(1).y_direction(), 0.0f);
    }
}

TEST(BattleRuntimeTest, ReceiveInputReturnsFalseForMissingRoom) {
    RoomManager room_manager;
    SessionManager session_manager(room_manager);
    BattleRuntime runtime(room_manager, session_manager,
                          [](const v1::ServerPacket&, const UdpEndpoint&) {});

    EXPECT_FALSE(runtime.receive_input("missing-room", 1001, 1.0f, 0.0f));
}

TEST(BattleRuntimeTest, EndRoomBroadcastsGameOverAndCleansRoom) {
    RoomManager room_manager;
    SessionManager session_manager(room_manager);
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    BattleRuntime runtime(room_manager, session_manager,
                          [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
                              sent_packets.emplace_back(packet, endpoint);
                          });

    ASSERT_EQ(room_manager.create_room({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_ids = {1001, 1002},
              }).status,
              CreateRoomStatus::OK);
    ASSERT_EQ(session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1001,
                  .conv = 1,
                  .endpoint = endpoint_with_port(7001),
              }).status,
              JoinSessionStatus::OK);
    ASSERT_EQ(session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1002,
                  .conv = 2,
                  .endpoint = endpoint_with_port(7002),
              }).status,
              JoinSessionStatus::OK);
    runtime.start_room("room-1");
    sent_packets.clear();

    auto result = runtime.end_room("room-1", "manual_end");

    EXPECT_EQ(result.status, EndRoomStatus::OK);
    EXPECT_EQ(result.message, "room ended");
    ASSERT_EQ(sent_packets.size(), 2);
    for (const auto& [packet, endpoint] : sent_packets) {
        ASSERT_EQ(packet.payload_case(), v1::ServerPacket::kGameOver);
        EXPECT_EQ(packet.game_over().room_name(), "room-1");
        EXPECT_EQ(packet.game_over().reason(), "manual_end");
        ASSERT_EQ(packet.game_over().player_ids_size(), 2);
        EXPECT_EQ(packet.game_over().player_ids(0), 1001);
        EXPECT_EQ(packet.game_over().player_ids(1), 1002);
    }
    EXPECT_FALSE(runtime.receive_input("room-1", 1001, 1.0f, 0.0f));
    EXPECT_EQ(session_manager.sessions_in_room("room-1").size(), 0);
    EXPECT_EQ(room_manager.active_rooms(), 0);
}

TEST(BattleRuntimeTest, EndRoomReturnsNotFoundForMissingRoom) {
    RoomManager room_manager;
    SessionManager session_manager(room_manager);
    BattleRuntime runtime(room_manager, session_manager,
                          [](const v1::ServerPacket&, const UdpEndpoint&) {});

    auto result = runtime.end_room("missing-room", "manual_end");

    EXPECT_EQ(result.status, EndRoomStatus::RoomNotFound);
}

}  // namespace
}  // namespace battle
