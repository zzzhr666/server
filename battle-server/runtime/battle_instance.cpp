#include "battle_instance.hpp"

#include <algorithm>
#include <array>
#include <random>
#include <ranges>
#include <utility>

namespace {
constexpr std::size_t RewardOptionCount = 3;

constexpr std::array<battle::BlessingID, 5> AllBlessingIDs{
    battle::BlessingID::BurnOnHit,
    battle::BlessingID::LifeSteal,
    battle::BlessingID::FreezeOnHit,
    battle::BlessingID::CriticalStrike,
    battle::BlessingID::ChainLightning,
};

bool contains_blessing(const std::vector<battle::BlessingID>& blessings, battle::BlessingID blessing_id) {
    return std::ranges::find(blessings, blessing_id) != blessings.end();
}
}

battle::BattleInstance::BattleInstance(BattleInstanceConfig config)
    : room_name_(std::move(config.room_name)),
      world_(config.world_bounds),
      state_(BattleState::Running),
      end_reason_(BattleEndReason::None),
      current_wave_(0),
      wave_config_(std::move(config.wave_config)),
      wave_planner_(),
      phase_(BattlePhase::Fighting),
      reward_selection_(SelectionTime),
      progression_config_(config.progression_config),
      reward_random_engine_(config.reward_random_seed.value_or(std::random_device{}())) {
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
        if (auto* progress = world_.player_progress().try_get(entity)) {
            progress->experience_to_next_level = experience_to_next_level_(progress->level);
        }
        player_entities_.emplace(config.player_ids[i], entity);
        entity_players_.emplace(entity, config.player_ids[i]);
        player_battle_stats_.try_emplace(config.player_ids[i]);
        player_blessings_.emplace(config.player_ids[i], PlayerBlessingState{.player_id = config.player_ids[i]});
    }
}

void battle::BattleInstance::tick(ecs::DeltaTime delta_time) {
    if (state_ == BattleState::Ended) {
        return;
    }
    if (phase_ == BattlePhase::RewardSelection) {
        tick_reward_selection_(delta_time);
        return;
    }
    tick_fighting_(delta_time);
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
    battle_world_snapshot.current_wave = current_wave_;
    battle_world_snapshot.phase = phase_;
    battle_world_snapshot.reward_selection_remaining = reward_selection_.remaining_seconds;
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
    for (auto [player_id, player_entity] : player_entities_) {
        auto progress = world_.player_progress().try_get(player_entity);
        if (!progress) {
            continue;
        }
        battle_world_snapshot.player_progress.emplace_back(player_id, progress->level, progress->experience,
                                                           progress->experience_to_next_level,
                                                           progress->pending_upgrade_choices);
    }

    for (auto& player_blessing : player_blessings_ | std::views::values) {
        battle_world_snapshot.player_blessings.emplace_back(player_blessing.player_id, player_blessing.blessings,
                                                            player_blessing.current_options);
    }
    std::ranges::sort(battle_world_snapshot.player_progress,
                      [](const PlayerProgressSnapshot& lhs, const PlayerProgressSnapshot& rhs) {
                          return lhs.player_id < rhs.player_id;
                      });

    std::ranges::sort(battle_world_snapshot.player_blessings,
                      [](const PlayerBlessingState& lhs, const PlayerBlessingState& rhs) {
                          return lhs.player_id < rhs.player_id;
                      });

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
        std::ranges::sort(player.kills, [](const MonsterKillCount& lhs, const MonsterKillCount& rhs) {
            return static_cast<int>(lhs.monster_kind) < static_cast<int>(rhs.monster_kind);
        });
        result.players.emplace_back(std::move(player));
    }
    std::ranges::sort(result.players, [](const PlayerSettlement& lhs, const PlayerSettlement& rhs) {
        return lhs.player_id < rhs.player_id;
    });
    return result;
}

std::optional<battle::ecs::PlayerProgress> battle::BattleInstance::player_progress(std::int64_t player_id) const {
    auto it = player_entities_.find(player_id);
    if (it == player_entities_.end()) {
        return std::nullopt;
    }
    const auto* progress = world_.player_progress().try_get(it->second);
    if (!progress) {
        return std::nullopt;
    }
    return *progress;
}

std::optional<battle::PlayerBlessingState> battle::BattleInstance::player_blessing_state(std::int64_t player_id) const {
    auto it = player_blessings_.find(player_id);
    return it == player_blessings_.end() ? std::nullopt : std::make_optional(it->second);
}

bool battle::BattleInstance::choose_blessing(std::int64_t player_id, int option_id) {
    if (phase_ != BattlePhase::RewardSelection) {
        return false;
    }
    auto blessing_it = player_blessings_.find(player_id);
    if (blessing_it == player_blessings_.end()) {
        return false;
    }
    auto entity_it = player_entities_.find(player_id);
    if (entity_it == player_entities_.end()) {
        return false;
    }

    auto progress = world_.player_progress().try_get(entity_it->second);
    if (!progress || progress->pending_upgrade_choices <= 0) {
        return false;
    }

    auto& blessing_state = blessing_it->second;
    auto option_it = std::ranges::find_if(blessing_state.current_options, [option_id](const BlessingOption& option) {
        return option.option_id == option_id;
    });
    if (option_it == blessing_state.current_options.end()) {
        return false;
    }

    add_or_level_up_blessing_(entity_it->second, blessing_state, option_it->blessing_id);

    progress->pending_upgrade_choices--;
    if (progress->pending_upgrade_choices > 0) {
        blessing_state.current_options = generate_blessing_options_(player_id);
    } else {
        blessing_state.current_options.clear();
    }

    return true;
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
        const auto player_id = killer_it->second;
        grant_experience_(player_id, experience_for_monster_kind_(event.monster_kind));
        auto& stats = player_battle_stats_[player_id];
        stats.total_kills++;
        stats.kills_by_kind[event.monster_kind]++;
    }
    world_.clear_kill_events();
}

void battle::BattleInstance::tick_fighting_(ecs::DeltaTime delta_time) {
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
        start_reward_selection_();
    }
}

void battle::BattleInstance::tick_reward_selection_(ecs::DeltaTime delta_time) {
    reward_selection_.remaining_seconds -= delta_time;
    if (reward_selection_.remaining_seconds.count() <= 0.0f) {
        apply_default_upgrade_choices_();
        start_next_wave_or_end_();
    }
}

void battle::BattleInstance::start_reward_selection_() {
    phase_ = BattlePhase::RewardSelection;
    reward_selection_.remaining_seconds = SelectionTime;

    for (auto& [player_id, blessing_state] : player_blessings_) {
        const auto progress = player_progress(player_id);
        if (!progress.has_value() || progress->pending_upgrade_choices <= 0) {
            blessing_state.current_options.clear();
            continue;
        }
        blessing_state.current_options = generate_blessing_options_(player_id);
    }
    // TODO: Generate upgrade choices and clear stale player selection state.
}

void battle::BattleInstance::apply_default_upgrade_choices_() {
    std::vector<std::int64_t> player_ids;
    player_ids.reserve(player_blessings_.size());
    for (const auto& player_id : player_blessings_ | std::views::keys) {
        player_ids.emplace_back(player_id);
    }
    for (const auto player_id : player_ids) {
        while (true) {
            auto blessing_state = player_blessing_state(player_id);
            auto progress = player_progress(player_id);
            if (!progress.has_value() || progress->pending_upgrade_choices <= 0) {
                break;
            }
            if (!blessing_state.has_value() || blessing_state->current_options.empty()) {
                break;
            }
            const auto option_id = blessing_state->current_options.front().option_id;
            if (!choose_blessing(player_id, option_id)) {
                break;
            }
        }
    }
}

void battle::BattleInstance::start_next_wave_or_end_() {
    if (current_wave_ >= wave_config_.waves.size()) {
        end_battle_(BattleEndReason::Victory);
        return;
    }
    phase_ = BattlePhase::Fighting;
    spawn_next_wave_();
}

void battle::BattleInstance::grant_experience_(std::int64_t player_id, int experience) {
    if (experience <= 0) {
        return;
    }
    auto it = player_entities_.find(player_id);
    if (it == player_entities_.end()) {
        return;
    }
    ecs::Entity entity = it->second;
    auto progress = world_.player_progress().try_get(entity);
    if (!progress) {
        return;
    }

    progress->experience += experience;
    while (progress->experience >= progress->experience_to_next_level) {
        progress->experience -= progress->experience_to_next_level;
        progress->pending_upgrade_choices++;
        progress->level++;
        progress->experience_to_next_level = experience_to_next_level_(progress->level);
    }
}

int battle::BattleInstance::experience_for_monster_kind_(MonsterKind kind) const {
    switch (kind) {
    case MonsterKind::Melee:
        return progression_config_.melee_experience;
    }

    return 0;
}

int battle::BattleInstance::experience_to_next_level_(int level) const {
    return progression_config_.base_experience_to_next_level +
        (level - 1) * progression_config_.experience_to_next_level_growth;
}

std::vector<battle::BlessingOption> battle::BattleInstance::generate_blessing_options_(std::int64_t player_id) {
    std::vector<BlessingID> selected;
    selected.reserve(RewardOptionCount);

    if (const auto blessing_state_it = player_blessings_.find(player_id); blessing_state_it != player_blessings_.end()) {
        for (const auto& blessing : blessing_state_it->second.blessings) {
            if (selected.size() >= RewardOptionCount) {
                break;
            }
            if (!contains_blessing(selected, blessing.blessing_id)) {
                selected.emplace_back(blessing.blessing_id);
            }
        }
    }

    auto candidates = AllBlessingIDs;
    std::ranges::shuffle(candidates, reward_random_engine_);
    for (const auto blessing_id : candidates) {
        if (selected.size() >= RewardOptionCount) {
            break;
        }
        if (!contains_blessing(selected, blessing_id)) {
            selected.emplace_back(blessing_id);
        }
    }

    std::vector<BlessingOption> options;
    options.reserve(selected.size());
    for (std::size_t i = 0; i < selected.size(); ++i) {
        options.emplace_back(BlessingOption{
            .option_id = static_cast<int>(i),
            .blessing_id = selected[i],
        });
    }
    return options;
}

void battle::BattleInstance::add_or_level_up_blessing_(ecs::Entity player_entity, PlayerBlessingState& blessing_state,
                                                       BlessingID blessing_id) {
    auto blessing_it = std::ranges::find_if(blessing_state.blessings, [blessing_id](const PlayerBlessing& blessing) {
        return blessing.blessing_id == blessing_id;
    });

    auto inventory = world_.blessing_inventories().try_get(player_entity);
    if (blessing_it != blessing_state.blessings.end()) {
        blessing_it->level++;
        if (inventory) {
            auto inventory_it = std::ranges::find_if(inventory->blessings,
                                                     [blessing_id](const ecs::BlessingStack& stack) {
                                                         return stack.blessing_id == blessing_id;
                                                     });
            if (inventory_it != inventory->blessings.end()) {
                inventory_it->level = blessing_it->level;
            } else {
                inventory->blessings.emplace_back(ecs::BlessingStack{
                    .blessing_id = blessing_id,
                    .level = blessing_it->level,
                });
            }
        }
        return;
    }
    blessing_state.blessings.emplace_back(PlayerBlessing{
        .blessing_id = blessing_id,
        .level = 1,
    });
    if (inventory) {
        inventory->blessings.emplace_back(ecs::BlessingStack{
            .blessing_id = blessing_id,
            .level = 1,
        });
    }
}
