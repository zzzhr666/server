#include "runtime/battle_runtime.hpp"

#include <algorithm>
#include <atomic>
#include <chrono>
#include <future>
#include <unordered_map>
#include <utility>
#include <vector>

#include <gtest/gtest.h>

#include "game/game_manager.hpp"
#include "platform/metrics.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"

#include <prometheus/registry.h>

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

struct RuntimeTestContext {
    prometheus::Registry registry;
    BattleMetrics metrics;
    RoomManager room_manager;
    SessionManager session_manager;

    RuntimeTestContext()
        : metrics(registry),
          room_manager(metrics),
          session_manager(room_manager, metrics) {}
};

TEST(BattleRuntimeTest, SoloRoomFirstJoinCompletesRoster) {
    RuntimeTestContext context;

    ASSERT_EQ(context.room_manager.create_room({
                  .room_name = "solo-room",
                  .token = "solo-token",
                  .player_ids = {1001},
              }).status,
              CreateRoomStatus::OK);

    const auto joined = context.session_manager.join({
        .room_name = "solo-room",
        .token = "solo-token",
        .player_id = 1001,
        .conv = 1,
        .endpoint = endpoint_with_port(7001),
    });

    EXPECT_EQ(joined.status, JoinSessionStatus::OK);
    EXPECT_TRUE(joined.all_players_joined);
    ASSERT_NE(joined.session, nullptr);
    EXPECT_EQ(joined.session->player_id(), 1001);
}

TEST(BattleRuntimeTest, ReceiveInputAndTickBroadcastsMovedSnapshot) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    BattleRuntime runtime(room_manager, session_manager, context.metrics,
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
        EXPECT_EQ(packet.snapshot().tick_rate(), DefaultBattleTickRate);
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

TEST(BattleRuntimeTest, TickBroadcastsConfiguredTickRate) {
    RuntimeTestContext context;
    std::vector<v1::ServerPacket> sent_packets;
    BattleRuntime runtime(
        context.room_manager, context.session_manager, context.metrics,
        [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint&) {
            sent_packets.push_back(packet);
        },
        {}, {}, 30);

    ASSERT_EQ(context.room_manager.create_room({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_ids = {1001},
              }).status,
              CreateRoomStatus::OK);
    ASSERT_EQ(context.session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1001,
                  .conv = 1,
                  .endpoint = endpoint_with_port(7001),
              }).status,
              JoinSessionStatus::OK);
    runtime.start_room("room-1");
    sent_packets.clear();

    runtime.tick(ecs::DeltaTime{0.0f});

    ASSERT_EQ(sent_packets.size(), 1);
    ASSERT_EQ(sent_packets.front().payload_case(), v1::ServerPacket::kSnapshot);
    EXPECT_EQ(sent_packets.front().snapshot().tick_rate(), 30);
}

TEST(BattleRuntimeTest, StartRoomFallsBackToDefaultTickRateForNegativeConfig) {
    RuntimeTestContext context;
    std::uint32_t captured_tick_rate = 0;
    BattleRuntime runtime(
        context.room_manager, context.session_manager, context.metrics,
        [](const v1::ServerPacket&, const UdpEndpoint&) {},
        [&captured_tick_rate](BattleInstanceConfig config) {
            captured_tick_rate = config.tick_rate;
            return std::make_unique<BattleInstance>(std::move(config));
        },
        {}, -1);

    ASSERT_EQ(context.room_manager.create_room({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_ids = {1001},
              }).status,
              CreateRoomStatus::OK);
    ASSERT_EQ(context.session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1001,
                  .conv = 1,
                  .endpoint = endpoint_with_port(7001),
              }).status,
              JoinSessionStatus::OK);

    runtime.start_room("room-1");

    EXPECT_EQ(captured_tick_rate, DefaultBattleTickRate);
}

TEST(BattleRuntimeTest, StartAdvancesWorldWithFixedDeltaTime) {
    RuntimeTestContext context;
    std::promise<v1::WorldSnapshot> first_snapshot_promise;
    auto first_snapshot_future = first_snapshot_promise.get_future();
    std::atomic<bool> snapshot_captured{false};
    constexpr int TickRate = 20;
    BattleRuntime runtime(
        context.room_manager, context.session_manager, context.metrics,
        [&first_snapshot_promise, &snapshot_captured](const v1::ServerPacket& packet, const UdpEndpoint&) {
            bool expected = false;
            if (packet.payload_case() == v1::ServerPacket::kSnapshot &&
                snapshot_captured.compare_exchange_strong(expected, true)) {
                first_snapshot_promise.set_value(packet.snapshot());
            }
        },
        [](BattleInstanceConfig config) {
            config.wave_config = WaveConfig{};
            return std::make_unique<BattleInstance>(std::move(config));
        },
        {}, TickRate);

    ASSERT_EQ(context.room_manager.create_room({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_ids = {1001},
              }).status,
              CreateRoomStatus::OK);
    ASSERT_EQ(context.session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1001,
                  .conv = 1,
                  .endpoint = endpoint_with_port(7001),
              }).status,
              JoinSessionStatus::OK);
    runtime.start_room("room-1");
    ASSERT_TRUE(runtime.receive_input("room-1", 1001, PlayerInput{
                                                   .move_x = 1.0f,
                                                   .move_y = 0.0f,
                                               }));

    runtime.start();
    const auto snapshot_status = first_snapshot_future.wait_for(std::chrono::seconds{1});
    runtime.stop();

    ASSERT_EQ(snapshot_status, std::future_status::ready);
    const auto snapshot = first_snapshot_future.get();
    ASSERT_EQ(snapshot.entities_size(), 1);
    EXPECT_FLOAT_EQ(snapshot.entities(0).x_position(),
                    -2.0f + ecs::DefaultPlayerMoveSpeed / static_cast<float>(TickRate));
    EXPECT_EQ(snapshot.server_tick(), 1);
}

TEST(BattleRuntimeTest, ReceiveInputAndTickBroadcastsProjectileSnapshot) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    BattleRuntime runtime(
        room_manager, session_manager, context.metrics,
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
                    },
                },
            };
            config.player_config_override = ecs::CreatePlayerConfig{
                .max_health = 100,
                .move_speed = 5.0f,
                .attack = ecs::AttackDefinition{
                    .kind = ecs::AttackKind::Projectile,
                    .damage = 30,
                    .range = 15.0f,
                    .cooldown_seconds = ecs::DeltaTime{0.5f},
                    .projectile_speed = 25.0f,
                },
            };
            return std::make_unique<BattleInstance>(std::move(config));
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
    const auto& snapshot = sent_packets.front().first.snapshot();
    EXPECT_EQ(snapshot.server_tick(), 1);
    ASSERT_EQ(snapshot.events_size(), 1);
    const auto& event = snapshot.events(0);
    EXPECT_EQ(event.event_id(), 1);
    ASSERT_EQ(event.payload_case(), v1::BattleEvent::kAttack);
    EXPECT_EQ(event.attack().attack_kind(), v1::ATTACK_KIND_PROJECTILE);
    EXPECT_NE(event.attack().attacker_entity(), 0);
    EXPECT_NE(event.attack().action_id(), 0);
    EXPECT_FLOAT_EQ(event.attack().x_direction(), 0.0f);
    EXPECT_FLOAT_EQ(event.attack().y_direction(), 1.0f);
    EXPECT_EQ(event.attack().start_tick(), 1);
    EXPECT_EQ(event.attack().active_start_tick(), 1);
    EXPECT_EQ(event.attack().active_end_tick(), 4);
    EXPECT_EQ(event.attack().recovery_end_tick(), 4);

    const auto& entities = snapshot.entities();
    const auto projectile_it = std::ranges::find_if(entities, [](const auto& entity) {
        return entity.kind() == v1::ENTITY_KIND_PROJECTILE;
    });
    ASSERT_NE(projectile_it, entities.end());
    EXPECT_EQ(projectile_it->current_health(), 0);
    EXPECT_EQ(projectile_it->max_health(), 0);

    runtime.tick(ecs::DeltaTime{0.0f});

    ASSERT_EQ(sent_packets.size(), 2);
    const auto& repeated_snapshot = sent_packets.back().first.snapshot();
    EXPECT_EQ(repeated_snapshot.server_tick(), 2);
    ASSERT_EQ(repeated_snapshot.events_size(), 1);
    const auto& repeated_attack = repeated_snapshot.events(0).attack();
    EXPECT_EQ(repeated_snapshot.events(0).event_id(), 1);
    EXPECT_EQ(repeated_attack.start_tick(), 1);
    EXPECT_EQ(repeated_attack.active_start_tick(), 1);
    EXPECT_EQ(repeated_attack.active_end_tick(), 4);
    EXPECT_EQ(repeated_attack.recovery_end_tick(), 4);
}

TEST(BattleRuntimeTest, TickBroadcastsRangedMonsterKind) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    BattleRuntime runtime(
        room_manager, session_manager, context.metrics,
        [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
            sent_packets.emplace_back(packet, endpoint);
        },
        [](BattleInstanceConfig config) {
            config.wave_config = WaveConfig{
                .waves = {
                    WaveDefinition{
                        .groups = {
                            WaveMonsterGroup{
                                .kind = MonsterKind::Ranged,
                                .count = 1,
                            },
                        },
                    },
                },
            };
            return std::make_unique<BattleInstance>(std::move(config));
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
    runtime.tick(ecs::DeltaTime{0.0f});

    ASSERT_EQ(sent_packets.size(), 1);
    const auto& entities = sent_packets.front().first.snapshot().entities();
    const auto ranged_monster_it = std::ranges::find_if(entities, [](const auto& entity) {
        return entity.kind() == v1::ENTITY_KIND_MONSTER && entity.monster_kind() == "ranged";
    });
    EXPECT_NE(ranged_monster_it, entities.end());
}

TEST(BattleRuntimeTest, ReceiveInputReturnsFalseForMissingRoom) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    BattleRuntime runtime(room_manager, session_manager, context.metrics,
                          [](const v1::ServerPacket&, const UdpEndpoint&) {});

    EXPECT_FALSE(runtime.receive_input("missing-room", 1001, PlayerInput{
                                                       .move_x = 1.0f,
                                                       .move_y = 0.0f,
                                                   }));
}

TEST(BattleRuntimeTest, StartRoomPassesConfiguredPlayerWeaponsToInstance) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::unordered_map<std::int64_t, std::pair<WeaponKind,GrowthLevels>> captured_weapons;
    BattleRuntime runtime(
        room_manager, session_manager, context.metrics,
        [](const v1::ServerPacket&, const UdpEndpoint&) {},
        [&captured_weapons](BattleInstanceConfig config) {
            captured_weapons = config.player_loadouts;
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
    EXPECT_EQ(captured_weapons.at(1001).first, WeaponKind::Dagger);
}

TEST(BattleRuntimeTest, EndRoomBroadcastsGameOverAndCleansRoom) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    std::vector<FinishedBattle> finished_battles;
    BattleRuntime runtime(room_manager, session_manager, context.metrics,
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

TEST(BattleRuntimeTest, TickSkipsSnapshotsForDisconnectedSessions) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    BattleRuntime runtime(room_manager, session_manager, context.metrics,
                          [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
                              sent_packets.emplace_back(packet, endpoint);
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

    const auto stale_at = std::chrono::steady_clock::now() + std::chrono::seconds{15};
    EXPECT_EQ(session_manager.mark_stale_sessions(stale_at, std::chrono::seconds{15}), 1);
    EXPECT_TRUE(session_manager.connected_sessions_in_room("room-1").empty());

    runtime.tick(ecs::DeltaTime{0.0f});

    EXPECT_TRUE(sent_packets.empty());
}

TEST(BattleRuntimeTest, EndRoomNotifiesOnlyConnectedSessionsButFinishesAllPlayers) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    std::vector<FinishedBattle> finished_battles;
    std::vector<std::int64_t> instance_player_ids;
    BattleRuntime runtime(
        room_manager, session_manager, context.metrics,
        [&sent_packets](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
            sent_packets.emplace_back(packet, endpoint);
        },
        [&instance_player_ids](BattleInstanceConfig config) {
            instance_player_ids = config.player_ids;
            return std::make_unique<BattleInstance>(std::move(config));
        },
        [&finished_battles](const FinishedBattle& finished_battle) {
            finished_battles.push_back(finished_battle);
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
    const auto stale_at = std::chrono::steady_clock::now() + std::chrono::seconds{15};
    ASSERT_EQ(session_manager.mark_stale_sessions(stale_at, std::chrono::seconds{15}), 1);
    ASSERT_EQ(session_manager.join({
                  .room_name = "room-1",
                  .token = "token-1",
                  .player_id = 1002,
                  .conv = 2,
                  .endpoint = endpoint_with_port(7002),
              }).status,
              JoinSessionStatus::OK);

    runtime.start_room("room-1");
    EXPECT_EQ(instance_player_ids, (std::vector<std::int64_t>{1001, 1002}));
    sent_packets.clear();

    ASSERT_EQ(runtime.end_room("room-1", "manual_end").status, EndRoomStatus::OK);

    ASSERT_EQ(sent_packets.size(), 1);
    EXPECT_EQ(sent_packets[0].second, endpoint_with_port(7002));
    ASSERT_EQ(sent_packets[0].first.payload_case(), v1::ServerPacket::kGameOver);
    ASSERT_EQ(sent_packets[0].first.game_over().player_ids_size(), 2);
    EXPECT_EQ(sent_packets[0].first.game_over().player_ids(0), 1001);
    EXPECT_EQ(sent_packets[0].first.game_over().player_ids(1), 1002);
    ASSERT_EQ(finished_battles.size(), 1);
    EXPECT_EQ(finished_battles[0].player_ids, (std::vector<std::int64_t>{1001, 1002}));
}

TEST(BattleRuntimeTest, AllDisconnectedTimeoutEndsRoomAndFinishesAllPlayers) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<FinishedBattle> finished_battles;
    BattleRuntime runtime(
        room_manager, session_manager, context.metrics,
        [](const v1::ServerPacket&, const UdpEndpoint&) {},
        {},
        [&finished_battles](const FinishedBattle& finished_battle) {
            finished_battles.push_back(finished_battle);
        },
        60,
        std::chrono::seconds{0},
        std::chrono::seconds{0});

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

    runtime.tick(ecs::DeltaTime{0.0f});
    EXPECT_TRUE(finished_battles.empty());
    runtime.tick(ecs::DeltaTime{0.0f});

    ASSERT_EQ(finished_battles.size(), 1);
    EXPECT_EQ(finished_battles[0].reason, "all_players_disconnected");
    EXPECT_EQ(finished_battles[0].player_ids, (std::vector<std::int64_t>{1001}));
    EXPECT_EQ(room_manager.active_rooms(), 0);
    EXPECT_TRUE(session_manager.sessions_in_room("room-1").empty());
}

TEST(BattleRuntimeTest, ReconnectedPlayerClearsAllDisconnectedTimeout) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<FinishedBattle> finished_battles;
    BattleRuntime runtime(
        room_manager, session_manager, context.metrics,
        [](const v1::ServerPacket&, const UdpEndpoint&) {},
        {},
        [&finished_battles](const FinishedBattle& finished_battle) {
            finished_battles.push_back(finished_battle);
        },
        60,
        std::chrono::hours{1},
        std::chrono::seconds{0});

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

    ASSERT_EQ(session_manager.mark_stale_sessions(std::chrono::steady_clock::now() + std::chrono::hours{2},
                                                  std::chrono::hours{1}),
              1);
    runtime.tick(ecs::DeltaTime{0.0f});
    ASSERT_TRUE(finished_battles.empty());

    const auto rejoined = session_manager.join({
        .room_name = "room-1",
        .token = "token-1",
        .player_id = 1001,
        .conv = 2,
        .endpoint = endpoint_with_port(7002),
    });
    ASSERT_EQ(rejoined.status, JoinSessionStatus::AlreadyJoined);
    ASSERT_EQ(rejoined.session->state(), BattleSessionState::Connected);
    EXPECT_EQ(rejoined.session->endpoint(), endpoint_with_port(7002));

    runtime.tick(ecs::DeltaTime{0.0f});
    runtime.tick(ecs::DeltaTime{0.0f});
    EXPECT_TRUE(finished_battles.empty());
    EXPECT_EQ(room_manager.active_rooms(), 1);
}

TEST(BattleRuntimeTest, TickBroadcastsGameOverAndCleansRoomWhenInstanceEnds) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    std::vector<std::pair<v1::ServerPacket, UdpEndpoint>> sent_packets;
    std::vector<FinishedBattle> finished_battles;
    BattleRuntime runtime(
        room_manager, session_manager, context.metrics,
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
    EXPECT_EQ(chosen_snapshot.current_wave(),2);
    EXPECT_EQ(chosen_snapshot.phase(),v1::BATTLE_PHASE_FIGHTING);
    ASSERT_EQ(chosen_snapshot.player_progress_size(), 1);
    EXPECT_EQ(chosen_snapshot.player_progress(0).pending_upgrade_choices(), 0);
    ASSERT_EQ(chosen_snapshot.player_blessings_size(), 1);
    ASSERT_EQ(chosen_snapshot.player_blessings(0).blessings_size(), 1);
    EXPECT_EQ(chosen_snapshot.player_blessings(0).blessings(0).blessing_id(), selected_blessing_id);
    EXPECT_EQ(chosen_snapshot.player_blessings(0).blessings(0).level(), 1);
    EXPECT_EQ(chosen_snapshot.player_blessings(0).current_options_size(), 0);



    ASSERT_TRUE(runtime.receive_input("room-1", 1001, PlayerInput{
                                                   .attack_requested = true,
                                               }));
    sent_packets.clear();
    runtime.tick(ecs::DeltaTime{0.5f});

    ASSERT_EQ(sent_packets.size(), 2);
    const auto& final_snapshot_packet = sent_packets[0].first;
    ASSERT_EQ(final_snapshot_packet.payload_case(), v1::ServerPacket::kSnapshot);
    const auto& final_snapshot = final_snapshot_packet.snapshot();
    const auto death_event = std::ranges::find_if(final_snapshot.events(), [](const auto& event) {
        return event.payload_case() == v1::BattleEvent::kDeath;
    });
    ASSERT_NE(death_event, final_snapshot.events().end());
    EXPECT_EQ(death_event->death().victim_kind(), v1::ENTITY_KIND_MONSTER);
    EXPECT_EQ(death_event->death().monster_kind(), "melee");

    const auto& game_over_packet = sent_packets[1].first;
    ASSERT_EQ(game_over_packet.payload_case(), v1::ServerPacket::kGameOver);
    EXPECT_EQ(game_over_packet.game_over().room_name(), "room-1");
    EXPECT_EQ(game_over_packet.game_over().reason(), "victory");
    ASSERT_EQ(game_over_packet.game_over().player_ids_size(), 1);
    EXPECT_EQ(game_over_packet.game_over().player_ids(0), 1001);
    ASSERT_EQ(game_over_packet.game_over().player_stats_size(), 1);
    EXPECT_EQ(game_over_packet.game_over().player_stats(0).player_id(), 1001);
    EXPECT_EQ(game_over_packet.game_over().player_stats(0).total_kills(), 2);
    ASSERT_EQ(game_over_packet.game_over().player_stats(0).kills_size(), 1);
    EXPECT_EQ(game_over_packet.game_over().player_stats(0).kills(0).monster_kind(), "melee");
    EXPECT_EQ(game_over_packet.game_over().player_stats(0).kills(0).count(), 2);
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
    EXPECT_EQ(player.total_kills, 2);
    ASSERT_EQ(player.kills.size(), 1);
    EXPECT_EQ(player.kills[0].monster_kind, MonsterKind::Melee);
    EXPECT_EQ(player.kills[0].count, 2);
}

TEST(BattleRuntimeTest, EndRoomReturnsNotFoundForMissingRoom) {
    RuntimeTestContext context;
    auto& room_manager = context.room_manager;
    auto& session_manager = context.session_manager;
    BattleRuntime runtime(room_manager, session_manager, context.metrics,
                          [](const v1::ServerPacket&, const UdpEndpoint&) {});

    auto result = runtime.end_room("missing-room", "manual_end");

    EXPECT_EQ(result.status, EndRoomStatus::RoomNotFound);
}

}  // namespace
}  // namespace battle
