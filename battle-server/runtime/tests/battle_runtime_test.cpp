#include "runtime/battle_runtime.hpp"

#include <unordered_map>
#include <utility>
#include <vector>

#include <gtest/gtest.h>

#include "game/game_manager.hpp"
#include "session/session_manager.hpp"

namespace battle {
namespace {

void expect_three_proto_blessing_options(const v1::PlayerBlessingStateSnapshot& blessing_state) {
    ASSERT_EQ(blessing_state.current_options_size(), 3);
    for (int i = 0; i < blessing_state.current_options_size(); ++i) {
        EXPECT_EQ(blessing_state.current_options(i).option_id(), i);
        for (int j = i + 1; j < blessing_state.current_options_size(); ++j) {
            EXPECT_NE(blessing_state.current_options(i).blessing_id(),
                      blessing_state.current_options(j).blessing_id());
        }
    }
}

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

    ASSERT_TRUE(runtime.receive_input("room-1", 1002, PlayerInput{
                                                   .move_x = 1.0f,
                                                   .move_y = 0.0f,
                                               }));
    runtime.tick(ecs::DeltaTime{1.0f});

    ASSERT_EQ(sent_packets.size(), 2);
    for (const auto& [packet, endpoint] : sent_packets) {
        ASSERT_EQ(packet.payload_case(), v1::ServerPacket::kSnapshot);
        EXPECT_EQ(packet.snapshot().room_name(), "room-1");
        ASSERT_GE(packet.snapshot().entities_size(), 2);
        EXPECT_EQ(packet.snapshot().entities(0).kind(), v1::ENTITY_KIND_PLAYER);
        EXPECT_EQ(packet.snapshot().entities(0).player_id(), 1001);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(0).x_position(), -2.0f);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(0).y_position(), 0.0f);
        EXPECT_EQ(packet.snapshot().entities(1).kind(), v1::ENTITY_KIND_PLAYER);
        EXPECT_EQ(packet.snapshot().entities(1).player_id(), 1002);
        EXPECT_FLOAT_EQ(packet.snapshot().entities(1).x_position(), 2.0f + ecs::DefaultPlayerMoveSpeed);
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

    EXPECT_FALSE(runtime.receive_input("missing-room", 1001, PlayerInput{
                                                       .move_x = 1.0f,
                                                       .move_y = 0.0f,
                                                   }));
}

TEST(BattleRuntimeTest, StartRoomPassesConfiguredPlayerWeaponsToInstance) {
    RoomManager room_manager;
    SessionManager session_manager(room_manager);
    std::unordered_map<std::int64_t, WeaponKind> captured_weapons;
    BattleRuntime runtime(
        room_manager, session_manager,
        [](const v1::ServerPacket&, const UdpEndpoint&) {},
        [&captured_weapons](BattleInstanceConfig config) {
            captured_weapons = config.player_weapons;
            return std::make_unique<BattleInstance>(std::move(config));
        });

    ASSERT_EQ(room_manager.create_room({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_ids = {1001},
                  .player_loadouts = {
                      PlayerLoadout{
                          .player_id = 1001,
                          .weapon = "dagger",
                      },
                  },
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

    runtime.start_room("room-1");

    ASSERT_TRUE(captured_weapons.contains(1001));
    EXPECT_EQ(captured_weapons.at(1001), WeaponKind::Dagger);
}

TEST(BattleRuntimeTest, EndRoomBroadcastsGameOverAndCleansRoom) {
    RoomManager room_manager;
    SessionManager session_manager(room_manager);
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    std::vector<FinishedBattle> finished_battles;
    BattleRuntime runtime(room_manager, session_manager,
                          [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
                              sent_packets.emplace_back(packet, endpoint);
                          },
                          {},
                          [&finished_battles](const FinishedBattle& finished_battle) {
                              finished_battles.emplace_back(finished_battle);
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
        ASSERT_EQ(packet.game_over().player_stats_size(), 2);
        EXPECT_EQ(packet.game_over().player_stats(0).player_id(), 1001);
        EXPECT_EQ(packet.game_over().player_stats(0).total_kills(), 0);
        EXPECT_EQ(packet.game_over().player_stats(0).kills_size(), 0);
        EXPECT_EQ(packet.game_over().player_stats(1).player_id(), 1002);
        EXPECT_EQ(packet.game_over().player_stats(1).total_kills(), 0);
        EXPECT_EQ(packet.game_over().player_stats(1).kills_size(), 0);
    }
    EXPECT_FALSE(runtime.receive_input("room-1", 1001, PlayerInput{
                                                   .move_x = 1.0f,
                                                   .move_y = 0.0f,
                                               }));
    EXPECT_EQ(session_manager.sessions_in_room("room-1").size(), 0);
    EXPECT_EQ(room_manager.active_rooms(), 0);
    ASSERT_EQ(finished_battles.size(), 1);
    EXPECT_EQ(finished_battles[0].room_name, "room-1");
    EXPECT_EQ(finished_battles[0].player_ids, (std::vector<std::int64_t>{1001, 1002}));
}

TEST(BattleRuntimeTest, TickBroadcastsGameOverAndCleansRoomWhenInstanceEnds) {
    RoomManager room_manager;
    SessionManager session_manager(room_manager);
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    std::vector<FinishedBattle> finished_battles;
    BattleRuntime runtime(
        room_manager, session_manager,
        [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
            sent_packets.emplace_back(packet, endpoint);
        },
        [](BattleInstanceConfig config) {
            config.wave_config = WaveConfig{
                .waves = {
                    WaveDefinition{
                        .groups = {
                            WaveMonsterGroup{
                                .kind = MonsterKind::Melee,
                                .count = 1,
                            },
                        },
                        .health_multiplier = 0.5f,
                        .move_speed_multiplier = 1.0f,
                    },
                },
            };
            config.player_config_override = ecs::CreatePlayerConfig{
                .max_health = 100,
                .move_speed = 5.0f,
                .attack = ecs::AttackDefinition{
                    .damage = 25,
                    .range = 20.0f,
                    .cooldown_seconds = ecs::DeltaTime{0.5f},
                },
            };
            config.progression_config = ProgressionConfig{
                .base_experience_to_next_level = 30,
                .experience_to_next_level_growth = 10,
                .melee_experience = 35,
            };
            return std::make_unique<BattleInstance>(std::move(config));
        },
        [&finished_battles](const FinishedBattle& finished_battle) {
            finished_battles.emplace_back(finished_battle);
        });

    ASSERT_EQ(room_manager.create_room({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_ids = {1001},
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
    runtime.start_room("room-1");
    sent_packets.clear();

    ASSERT_TRUE(runtime.receive_input("room-1", 1001, PlayerInput{
                                                   .attack_requested = true,
                                               }));
    runtime.tick(ecs::DeltaTime{0.0f});

    ASSERT_EQ(sent_packets.size(), 1);
    EXPECT_EQ(sent_packets[0].first.payload_case(), v1::ServerPacket::kSnapshot);
    const auto& snapshot = sent_packets[0].first.snapshot();
    EXPECT_EQ(snapshot.current_wave(), 1);
    EXPECT_EQ(snapshot.phase(), v1::BATTLE_PHASE_REWARD_SELECTION);
    EXPECT_FLOAT_EQ(snapshot.reward_selection_remaining_seconds(), SelectionTime.count());
    ASSERT_EQ(snapshot.player_progress_size(), 1);
    EXPECT_EQ(snapshot.player_progress(0).player_id(), 1001);
    EXPECT_EQ(snapshot.player_progress(0).level(), 2);
    EXPECT_EQ(snapshot.player_progress(0).experience(), 5);
    EXPECT_EQ(snapshot.player_progress(0).experience_to_next_level(), 40);
    EXPECT_EQ(snapshot.player_progress(0).pending_upgrade_choices(), 1);
    ASSERT_EQ(snapshot.player_blessings_size(), 1);
    EXPECT_EQ(snapshot.player_blessings(0).player_id(), 1001);
    expect_three_proto_blessing_options(snapshot.player_blessings(0));
    const auto selected_blessing_id = snapshot.player_blessings(0).current_options(0).blessing_id();

    ASSERT_TRUE(runtime.choose_blessing("room-1", 1001, snapshot.player_blessings(0).current_options(0).option_id()));
    sent_packets.clear();
    runtime.tick(ecs::DeltaTime{0.0f});

    ASSERT_EQ(sent_packets.size(), 1);
    ASSERT_EQ(sent_packets[0].first.payload_case(), v1::ServerPacket::kSnapshot);
    const auto& chosen_snapshot = sent_packets[0].first.snapshot();
    ASSERT_EQ(chosen_snapshot.player_progress_size(), 1);
    EXPECT_EQ(chosen_snapshot.player_progress(0).pending_upgrade_choices(), 0);
    ASSERT_EQ(chosen_snapshot.player_blessings_size(), 1);
    ASSERT_EQ(chosen_snapshot.player_blessings(0).blessings_size(), 1);
    EXPECT_EQ(chosen_snapshot.player_blessings(0).blessings(0).blessing_id(), selected_blessing_id);
    EXPECT_EQ(chosen_snapshot.player_blessings(0).blessings(0).level(), 1);
    EXPECT_EQ(chosen_snapshot.player_blessings(0).current_options_size(), 0);

    EXPECT_TRUE(runtime.receive_input("room-1", 1001, PlayerInput{
                                                   .move_x = 1.0f,
                                                   .move_y = 0.0f,
                                               }));
    EXPECT_EQ(session_manager.sessions_in_room("room-1").size(), 1);
    EXPECT_EQ(room_manager.active_rooms(), 1);

    sent_packets.clear();
    runtime.tick(SelectionTime);

    ASSERT_EQ(sent_packets.size(), 1);
    const auto& packet = sent_packets[0].first;
    ASSERT_EQ(packet.payload_case(), v1::ServerPacket::kGameOver);
    EXPECT_EQ(packet.game_over().room_name(), "room-1");
    EXPECT_EQ(packet.game_over().reason(), "victory");
    ASSERT_EQ(packet.game_over().player_ids_size(), 1);
    EXPECT_EQ(packet.game_over().player_ids(0), 1001);
    ASSERT_EQ(packet.game_over().player_stats_size(), 1);
    EXPECT_EQ(packet.game_over().player_stats(0).player_id(), 1001);
    EXPECT_EQ(packet.game_over().player_stats(0).total_kills(), 1);
    ASSERT_EQ(packet.game_over().player_stats(0).kills_size(), 1);
    EXPECT_EQ(packet.game_over().player_stats(0).kills(0).monster_kind(), "melee");
    EXPECT_EQ(packet.game_over().player_stats(0).kills(0).count(), 1);
    EXPECT_FALSE(runtime.receive_input("room-1", 1001, PlayerInput{
                                                   .move_x = 1.0f,
                                                   .move_y = 0.0f,
                                               }));
    EXPECT_EQ(session_manager.sessions_in_room("room-1").size(), 0);
    EXPECT_EQ(room_manager.active_rooms(), 0);
    ASSERT_EQ(finished_battles.size(), 1);
    EXPECT_EQ(finished_battles[0].room_name, "room-1");
    EXPECT_EQ(finished_battles[0].player_ids, (std::vector<std::int64_t>{1001}));
    EXPECT_EQ(finished_battles[0].settlement.reason, BattleEndReason::Victory);
    ASSERT_EQ(finished_battles[0].settlement.players.size(), 1);
    const auto& player = finished_battles[0].settlement.players[0];
    EXPECT_EQ(player.player_id, 1001);
    EXPECT_EQ(player.total_kills, 1);
    ASSERT_EQ(player.kills.size(), 1);
    EXPECT_EQ(player.kills[0].monster_kind, MonsterKind::Melee);
    EXPECT_EQ(player.kills[0].count, 1);
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
