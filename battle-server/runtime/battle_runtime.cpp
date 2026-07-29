#include "battle_runtime.hpp"

#include <utility>
#include <vector>
#include <cstdint>
#include <chrono>

#include "battle_instance.hpp"
#include "game/game_manager.hpp"
#include "gameplay/weapon.hpp"
#include "net/packet_codec.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"

namespace {
    battle::v1::EntityKind to_proto_entity_kind(battle::ecs::EntityKind entity_kind) {
        switch (entity_kind) {
        case battle::ecs::EntityKind::Player:
            return battle::v1::ENTITY_KIND_PLAYER;
        case battle::ecs::EntityKind::Monster:
            return battle::v1::ENTITY_KIND_MONSTER;
        case battle::ecs::EntityKind::Unknown:
        default:
            return battle::v1::ENTITY_KIND_UNSPECIFIED;
        }
    }


    std::string battle_end_reason_to_string(battle::BattleEndReason reason) {
        switch (reason) {
        case battle::BattleEndReason::Defeat: {
            return "defeat";
        }
        case battle::BattleEndReason::Victory: {
            return "victory";
        }
        default: {
            return "unknown";
        }
        }
    }

    std::vector<battle::PacketPlayerBattleStats> to_packet_player_stats(const battle::BattleSettlement& settlement) {
        std::vector<battle::PacketPlayerBattleStats> result;
        result.reserve(settlement.players.size());
        for (const auto& player : settlement.players) {
            battle::PacketPlayerBattleStats player_stat{
                .player_id = player.player_id,
                .total_kills = player.total_kills,
            };
            player_stat.kills.reserve(player.kills.size());
            for (const auto& kill : player.kills) {
                player_stat.kills.push_back(battle::PacketMonsterKillCount{
                    .monster_kind = kill.monster_kind,
                    .count = kill.count,
                });
            }
            result.emplace_back(std::move(player_stat));
        }
        return result;
    }

    battle::v1::BattlePhase to_proto_battle_phase(battle::BattlePhase phase) {
        switch (phase) {
        case battle::BattlePhase::Fighting: {
            return battle::v1::BattlePhase::BATTLE_PHASE_FIGHTING;
        }
        case battle::BattlePhase::RewardSelection: {
            return battle::v1::BattlePhase::BATTLE_PHASE_REWARD_SELECTION;
        }
        default: {
            return battle::v1::BattlePhase::BATTLE_PHASE_UNSPECIFIED;
        }
        }
    }

    battle::v1::BlessingId to_proto_blessing_id(battle::BlessingID blessing_id) {
        switch (blessing_id) {
        case battle::BlessingID::BurnOnHit: {
            return battle::v1::BLESSING_ID_BURN_ON_HIT;
        }
        case battle::BlessingID::LifeSteal: {
            return battle::v1::BLESSING_ID_LIFE_STEAL;
        }
        case battle::BlessingID::FreezeOnHit: {
            return battle::v1::BLESSING_ID_FREEZE_ON_HIT;
        }
        case battle::BlessingID::CriticalStrike: {
            return battle::v1::BLESSING_ID_CRITICAL_STRIKE;
        }
        case battle::BlessingID::ChainLightning: {
            return battle::v1::BLESSING_ID_CHAIN_LIGHTNING;
        }
        default: {
            return battle::v1::BLESSING_ID_UNSPECIFIED;
        }
        }
    }


    battle::v1::ServerPacket make_snapshot(const std::string& room_name, const battle::BattleWorldSnapshot& snapshot) {
        battle::v1::ServerPacket packet;
        auto send_pkg = packet.mutable_snapshot();
        send_pkg->set_room_name(room_name);
        send_pkg->set_current_wave(static_cast<std::int32_t>(snapshot.current_wave));
        send_pkg->set_phase(to_proto_battle_phase(snapshot.phase));
        send_pkg->set_reward_selection_remaining_seconds(snapshot.reward_selection_remaining.count());
        for (const auto& entity : snapshot.entities) {
            auto entity_snapshot = send_pkg->add_entities();
            entity_snapshot->set_entity(entity.entity);
            entity_snapshot->set_kind(to_proto_entity_kind(entity.kind));
            entity_snapshot->set_player_id(entity.player_id);
            entity_snapshot->set_x_position(entity.x_position);
            entity_snapshot->set_y_position(entity.y_position);
            entity_snapshot->set_x_direction(entity.x_direction);
            entity_snapshot->set_y_direction(entity.y_direction);
            entity_snapshot->set_current_health(entity.current_health);
            entity_snapshot->set_max_health(entity.max_health);
        }

        for (const auto& progress : snapshot.player_progress) {
            auto proto_progress = send_pkg->add_player_progress();
            proto_progress->set_player_id(progress.player_id);
            proto_progress->set_experience(progress.experience);
            proto_progress->set_level(progress.level);
            proto_progress->set_experience_to_next_level(progress.experience_to_next_level);
            proto_progress->set_pending_upgrade_choices(progress.pending_upgrade_choices);
        }

        for (const auto& player_blessing : snapshot.player_blessings) {
            auto proto_blessings = send_pkg->add_player_blessings();
            proto_blessings->set_player_id(player_blessing.player_id);
            for (const auto& blessing : player_blessing.blessings) {
                auto proto_blessing = proto_blessings->add_blessings();
                proto_blessing->set_level(blessing.level);
                proto_blessing->set_blessing_id(to_proto_blessing_id(blessing.blessing_id));
            }

            for (const auto& option : player_blessing.current_options) {
                auto proto_option = proto_blessings->add_current_options();
                proto_option->set_option_id(option.option_id);
                proto_option->set_blessing_id(to_proto_blessing_id(option.blessing_id));
            }
        }
        return packet;
    }
}

battle::BattleRuntime::BattleRuntime(RoomManager& room_manager, SessionManager& session_manager,
                                     SendPacketCallback send_packet_callback, BattleInstanceFactory factory,
                                     FinishMatchCallback finish_match_callback)
    : room_manager_(room_manager), session_manager_(session_manager),
      send_packet_(std::move(send_packet_callback)), finish_match_callback_(std::move(finish_match_callback)),
      running_(false), instance_factory_(std::move(factory)) {
    if (!instance_factory_) {
        instance_factory_ = [](BattleInstanceConfig config) {
            return std::make_unique<BattleInstance>(std::move(config));
        };
    }
}

battle::BattleRuntime::~BattleRuntime() {
    stop();
}

void battle::BattleRuntime::start_room(const std::string& room_name) {
    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (starting_rooms_.contains(room_name) || instances_.contains(room_name)) {
            return;
        }
        starting_rooms_.insert(room_name);
    }

    auto sessions = session_manager_.sessions_in_room(room_name);
    auto configured_loadouts = room_manager_.player_loadouts(room_name);

    std::vector<std::int64_t> player_ids;
    player_ids.reserve(sessions.size());
    for (const auto& session : sessions) {
        player_ids.push_back(session->player_id());
    }
    std::unordered_map<std::int64_t, WeaponKind> player_weapons;
    player_weapons.reserve(configured_loadouts.size());
    for (const auto& loadout : configured_loadouts) {
        auto weapon_kind = weapon_kind_from_string(loadout.weapon);
        if (!weapon_kind.has_value()) {
            continue;
        }
        player_weapons.emplace(loadout.player_id, weapon_kind.value());
    }

    auto instance = instance_factory_(BattleInstanceConfig{
        .room_name = room_name,
        .player_ids = player_ids,
        .player_weapons = std::move(player_weapons),
    });

    auto game_start_packet = make_game_start(room_name, player_ids);

    {
        std::lock_guard<std::mutex> lock(mutex_);
        starting_rooms_.erase(room_name);
        instances_.emplace(room_name, std::move(instance));
    }

    for (const auto& session : sessions) {
        send_packet_(game_start_packet, session->endpoint());
    }
}

void battle::BattleRuntime::tick(ecs::DeltaTime delta_time) {
    std::vector<v1::ServerPacket> packets;
    std::vector<UdpEndpoint> endpoints;
    std::vector<std::pair<std::string, BattleEndReason>> end_state;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        for (auto& [room_name, instance] : instances_) {
            instance->tick(delta_time);
            if (instance->ended()) {
                auto reason = instance->end_reason();
                end_state.emplace_back(room_name, reason);
                continue;
            }
            auto snapshot = instance->snapshot();
            const auto packet = make_snapshot(room_name, snapshot);
            auto sessions = session_manager_.sessions_in_room(room_name);
            for (auto& session : sessions) {
                packets.emplace_back(packet);
                endpoints.push_back(session->endpoint());
            }
        }
    }

    for (const auto& [room_name,reason] : end_state) {
        end_room(room_name, battle_end_reason_to_string(reason));
    }

    for (std::size_t i = 0; i < packets.size(); i++) {
        send_packet_(packets[i], endpoints[i]);
    }
}

bool battle::BattleRuntime::receive_input(const std::string& room_name, std::int64_t player_id, PlayerInput input) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = instances_.find(room_name);
    if (it == instances_.end()) {
        return false;
    }
    return it->second->receive_input(player_id, input);
}

bool battle::BattleRuntime::choose_blessing(const std::string& room_name, std::int64_t player_id, int option_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = instances_.find(room_name);
    return it == instances_.end() ? false : it->second->choose_blessing(player_id, option_id);
}

void battle::BattleRuntime::start() {
    bool expected = false;
    if (!running_.compare_exchange_strong(expected, true)) {
        return;
    }
    tick_thread_ = std::thread([this]() {
        using clock = std::chrono::steady_clock;
        constexpr auto tick_interval = std::chrono::milliseconds(50);
        auto last_tick = clock::now();
        while (running_) {
            auto now = clock::now();
            const ecs::DeltaTime delta = now - last_tick;
            last_tick = now;
            tick(delta);

            std::this_thread::sleep_for(tick_interval);
        }
    });
}

void battle::BattleRuntime::stop() {
    running_ = false;
    if (tick_thread_.joinable()) {
        tick_thread_.join();
    }
}

battle::EndRoomResult battle::BattleRuntime::end_room(const std::string& room_name, const std::string& reason) {
    v1::ServerPacket packet;
    std::vector<UdpEndpoint> endpoints;
    std::vector<std::int64_t> player_ids;
    BattleSettlement settlement;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = instances_.find(room_name);
        if (it == instances_.end()) {
            return {
                .status = EndRoomStatus::RoomNotFound,
                .message = "unable to find instance",
            };
        }
        settlement = it->second->settlement();
        auto sessions = session_manager_.sessions_in_room(room_name);

        player_ids.reserve(sessions.size());
        for (const auto& session : sessions) {
            player_ids.push_back(session->player_id());
            endpoints.push_back(session->endpoint());
        }
        packet = make_game_over(room_name, player_ids, reason, to_packet_player_stats(settlement));
        instances_.erase(it);
    }
    for (auto& endpoint : endpoints) {
        send_packet_(packet, endpoint);
    }

    session_manager_.remove_room(room_name);
    room_manager_.close_room(room_name);
    FinishedBattle finished_battle{
        .room_name = room_name,
        .player_ids = player_ids,
        .settlement = settlement,
    };
    if (finish_match_callback_) {
        finish_match_callback_(finished_battle);
    }
    return {
        .status = EndRoomStatus::OK,
        .message = "room ended",
    };
}
