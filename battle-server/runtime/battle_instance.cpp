#include "battle_instance.hpp"

#include <algorithm>
#include <utility>


battle::BattleInstance::BattleInstance(BattleInstanceConfig config)
    : room_name_(std::move(config.room_name)),
      world_(config.world_bounds),
      state_(BattleState::Running),
      end_reason_(BattleEndReason::None),
      current_wave_(0),
      wave_config_(std::move(config.wave_config)),
      wave_planner_() {
    for (std::size_t i = 0; i < config.player_ids.size(); ++i) {
        if (player_entities_.contains(config.player_ids[i])) {
            continue;
        }
        auto spawn_config = spawn_planner_.player_spawn(i);

        auto weapon_kind = WeaponKind::Sword;
        if (auto it = config.player_weapons.find(config.player_ids[i]); it != config.player_weapons.end()) {
            weapon_kind = it->second;
        }
        auto weapon = weapon_definition(weapon_kind);
        spawn_config.attack = weapon.attack;
        if (config.player_config_override.has_value()) {
            auto override_config = config.player_config_override.value();
            override_config.position = spawn_config.position;
            spawn_config = override_config;
        }
        auto entity = world_.create_player(spawn_config);
        player_entities_.emplace(config.player_ids[i], entity);
        entity_players_.emplace(entity, config.player_ids[i]);
        player_battle_stats_.try_emplace(config.player_ids[i]);
    }
}

void battle::BattleInstance::tick(ecs::DeltaTime delta_time) {
    if (state_ == BattleState::Ended) {
        return;
    }
    if (current_wave_ == 0) {
        spawn_next_wave_();
    }
    world_.tick(delta_time);
    consume_kill_events_();
    if (!world_.has_living_players()) {
        end_battle_(BattleEndReason::Defeat);
        return;
    }
    if (!world_.has_living_monsters()) {
        if (current_wave_ >= wave_config_.waves.size()) {
            end_battle_(BattleEndReason::Victory);
            return;
        }
        spawn_next_wave_();
    }
}

bool battle::BattleInstance::receive_input(std::int64_t player_id, PlayerInput input) {
    auto it = player_entities_.find(player_id);
    if (it == player_entities_.end()) {
        return false;
    }
    return world_.set_player_command(it->second, ecs::PlayerCommand{
                                         .move_x = input.move_x,
                                         .move_y = input.move_y,
                                         .attack_requested = input.attack_requested,
                                         .dash_requested = input.dash_requested,
                                     });
}

battle::BattleWorldSnapshot battle::BattleInstance::snapshot() const {
    auto world_snapshot = world_.snapshot();

    BattleWorldSnapshot battle_world_snapshot;
    for (auto& snapshot : world_snapshot.entities) {
        std::int64_t player_id = 0;
        if (snapshot.kind == ecs::EntityKind::Player) {
            if (auto it = entity_players_.find(snapshot.entity); it != entity_players_.end()) {
                player_id = it->second;
            }
        }
        battle_world_snapshot.entities.emplace_back(snapshot.entity, snapshot.kind, player_id, snapshot.x_position,
                                                    snapshot.y_position, snapshot.x_direction, snapshot.y_direction,
                                                    snapshot.current_health, snapshot.max_health);
    }
    return battle_world_snapshot;
}

battle::BattleSettlement battle::BattleInstance::settlement() const {
    BattleSettlement result{
        .reason = end_reason_,
    };
    result.players.reserve(player_battle_stats_.size());
    for (const auto& [player_id, stats] : player_battle_stats_) {
        PlayerSettlement player{
            .player_id = player_id,
            .total_kills = stats.total_kills,
        };
        player.kills.reserve(stats.kills_by_kind.size());
        for (const auto& [monster_kind, count] : stats.kills_by_kind) {
            player.kills.push_back(MonsterKillCount{
                .monster_kind = monster_kind,
                .count = count,
            });
        }
        std::sort(player.kills.begin(), player.kills.end(), [](const MonsterKillCount& lhs, const MonsterKillCount& rhs) {
            return static_cast<int>(lhs.monster_kind) < static_cast<int>(rhs.monster_kind);
        });
        result.players.emplace_back(std::move(player));
    }
    std::sort(result.players.begin(), result.players.end(), [](const PlayerSettlement& lhs, const PlayerSettlement& rhs) {
        return lhs.player_id < rhs.player_id;
    });
    return result;
}

void battle::BattleInstance::spawn_next_wave_() {
    if (current_wave_ >= wave_config_.waves.size()) {
        return;
    }
    auto monster_configs = wave_planner_.plan_wave(wave_config_.waves[current_wave_]);
    for (const auto& monster_config : monster_configs) {
        world_.create_monster(monster_config);
    }
    ++current_wave_;
}

void battle::BattleInstance::end_battle_(BattleEndReason reason) {
    state_ = BattleState::Ended;
    end_reason_ = reason;
}

void battle::BattleInstance::consume_kill_events_() {
    for (const auto& event : world_.kill_events()) {
        auto killer_it = entity_players_.find(event.killer);
        if (killer_it == entity_players_.end()) {
            continue;
        }
        auto& stats = player_battle_stats_[killer_it->second];
        stats.total_kills++;
        stats.kills_by_kind[event.monster_kind]++;
    }
    world_.clear_kill_events();
}
