#include "battle_instance.hpp"

#include <algorithm>
#include <array>
#include <chrono>
#include <cmath>
#include <random>
#include <ranges>
#include <utility>

#include "gameplay/room_encounter_validator.hpp"
#include "gameplay/room_graph_validator.hpp"
#include "gameplay/room_layout_catalog_validator.hpp"
#include "gameplay/room_layout_validator.hpp"

namespace {
    constexpr std::size_t RewardOptionCount = 3;

    constexpr std::array<battle::BlessingID, 5> AllBlessingIDs{
        battle::BlessingID::BurnOnHit,
        battle::BlessingID::LifeSteal,
        battle::BlessingID::FreezeOnHit,
        battle::BlessingID::CriticalStrike,
        battle::BlessingID::ChainLightning,
    };

    bool valid_room_configuration(const battle::BattleInstanceConfig& config) {
        bool valid = battle::validate_room_graph(config.dungeon_room_graph).empty() &&
            battle::validate_room_encounters(config.dungeon_room_graph).empty() &&
            battle::validate_room_layout_catalog(config.room_layout_catalog).empty() &&
            battle::validate_room_layout_references(config.dungeon_room_graph, config.room_layout_catalog).empty();
        for (const auto& layout : config.room_layout_catalog.layouts) {
            valid &= battle::validate_room_layout(layout).empty();
        }
        return valid;
    }
}

battle::BattleInstance::BattleInstance(BattleInstanceConfig config)
    : room_name_(std::move(config.room_name)),
      world_(config.world_bounds),
      state_(BattleState::Running),
      end_reason_(BattleEndReason::None),
      reward_selection_(SelectionTime),
      progression_config_(config.progression_config),
      reward_random_engine_(config.reward_random_seed.value_or(std::random_device{}())),
      tick_rate_(config.tick_rate == 0 ? DefaultBattleTickRate : config.tick_rate),
      dungeon_room_graph_(std::move(config.dungeon_room_graph)),
      room_layout_catalog_(std::move(config.room_layout_catalog)),
      room_runtime_(dungeon_room_graph_, room_layout_catalog_) {
    // 创建玩家时将局外 loadout 固化为 ECS 基础属性。之后每个 tick 只使用内存世界，
    // 避免局外数据在同一场战斗中变化而破坏对局一致性。
    for (std::size_t i = 0; i < config.player_ids.size(); ++i) {
        if (player_entities_.contains(config.player_ids[i])) {
            continue;
        }
        auto spawn_config = spawn_planner_.player_spawn(i);

        auto hero_kind = HeroKind::Fire;
        GrowthLevels growth_lvl;
        if (auto it = config.player_loadouts.find(config.player_ids[i]); it != config.player_loadouts.end()) {
            hero_kind = it->second.first;
            growth_lvl = it->second.second;
        }

        auto hero = hero_definition(hero_kind);
        spawn_config.attack = hero.attack;
        if (config.player_config_override.has_value()) {
            auto override_config = config.player_config_override.value();
            override_config.position = spawn_config.position;
            spawn_config = override_config;
        }
        spawn_config = apply_growth(spawn_config, growth_lvl);
        auto entity = world_.create_player(spawn_config);
        if (entity == ecs::NullEntity) {
            initialization_failed_ = true;
            break;
        }
        player_heroes_.emplace(config.player_ids[i], hero_kind);
        if (auto* progress = world_.registry().try_get<ecs::PlayerProgress>(entity)) {
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
    ++server_tick_;
    discard_expired_combat_events_();

    if (room_state() == RoomFlowState::ChoosingBlessing) {
        tick_blessing_selection_(delta_time);
        return;
    }
    if (room_state() != RoomFlowState::Fighting) {
        return;
    }
    tick_fighting_(delta_time);
    combat_elapsed_time_ += delta_time;
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
    battle_world_snapshot.reward_selection_remaining = reward_selection_.remaining_seconds;
    battle_world_snapshot.server_tick = server_tick_;
    battle_world_snapshot.current_room_id = current_room_id();
    battle_world_snapshot.room_state = room_state();
    if (const auto* current_room = dungeon_room_graph_.find_room(current_room_id())) {
        battle_world_snapshot.current_room_layout_id = current_room->layout_id;
        for (const auto next_room_id : current_room->next_room_ids) {
            battle_world_snapshot.available_room_exit_ids.emplace_back(next_room_id);
        }
    }
    if (room_state() == RoomFlowState::ChoosingExit) {
        for (const auto& [player_id, chosen_room_id] : room_exit_choices_) {
            const auto entity_it = player_entities_.find(player_id);
            if (entity_it == player_entities_.end()) {
                continue;
            }
            if (world_.is_living_player(entity_it->second)) {
                battle_world_snapshot.player_room_exit_choices.emplace_back(player_id, chosen_room_id);
            }
        }
    }

    for (auto& snapshot : world_snapshot.entities) {
        std::int64_t player_id = 0;
        std::optional<HeroKind> hero;
        if (snapshot.kind == ecs::EntityKind::Player) {
            if (auto it = entity_players_.find(snapshot.entity); it != entity_players_.end()) {
                player_id = it->second;
                if (auto hero_it = player_heroes_.find(player_id); hero_it != player_heroes_.end()) {
                    hero = hero_it->second;
                }
            }
        }
        battle_world_snapshot.entities.emplace_back(snapshot.entity, snapshot.kind, player_id, hero,
                                                    snapshot.position, snapshot.direction,
                                                    snapshot.current_health, snapshot.max_health,
                                                    snapshot.monster_kind);
    }
    for (auto [player_id, player_entity] : player_entities_) {
        auto progress = world_.registry().try_get<ecs::PlayerProgress>(player_entity);
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
    for (auto& event : pending_battle_events_) {
        battle_world_snapshot.events.emplace_back(event.event);
    }
    std::ranges::sort(battle_world_snapshot.player_progress,
                      [](const PlayerProgressSnapshot& lhs, const PlayerProgressSnapshot& rhs) {
                          return lhs.player_id < rhs.player_id;
                      });

    std::ranges::sort(battle_world_snapshot.player_blessings,
                      [](const PlayerBlessingState& lhs, const PlayerBlessingState& rhs) {
                          return lhs.player_id < rhs.player_id;
                      });
    battle_world_snapshot.tick_rate = tick_rate_;
    return battle_world_snapshot;
}

battle::BattleSettlement battle::BattleInstance::settlement() const {
    BattleSettlement result{
        .reason = end_reason_,
        .combat_duration_ms = std::chrono::duration_cast<std::chrono::milliseconds>(combat_elapsed_time_).count()
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
    const auto* progress = world_.registry().try_get<ecs::PlayerProgress>(it->second);
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
    if (room_state() != RoomFlowState::ChoosingBlessing) {
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

    auto progress = world_.registry().try_get<ecs::PlayerProgress>(entity_it->second);
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

    // 候选仅在本轮 RewardSelection 有效。应用后立即减少待选次数，剩余次数需要
    // 重新抽取候选，避免客户端重复使用已消费 option_id。
    add_or_level_up_blessing_(entity_it->second, blessing_state, option_it->blessing_id);

    progress->pending_upgrade_choices--;
    if (progress->pending_upgrade_choices > 0) {
        blessing_state.current_options = generate_blessing_options_(player_id);
    } else {
        blessing_state.current_options.clear();
    }

    return true;
}

bool battle::BattleInstance::select_room_exit(std::int64_t player_id, DungeonRoomID next_room_id) {
    if (ended() || room_state() != RoomFlowState::ChoosingExit) {
        return false;
    }
    const auto player_it = player_entities_.find(player_id);
    if (player_it == player_entities_.end() || !world_.is_living_player(player_it->second)) {
        return false;
    }

    const auto* current_room = dungeon_room_graph_.find_room(current_room_id());
    if (current_room == nullptr || std::ranges::find(current_room->next_room_ids, next_room_id) ==
        current_room->next_room_ids.end()) {
        return false;
    }

    room_exit_choices_[player_id] = next_room_id;
    for (const auto& [living_player_id, entity] : player_entities_) {
        if (!world_.is_living_player(entity)) {
            continue;
        }
        const auto choice_it = room_exit_choices_.find(living_player_id);
        if (choice_it == room_exit_choices_.end() || choice_it->second != next_room_id) {
            return true;
        }
    }
    if (!room_runtime_.select_exit(next_room_id)) {
        end_battle_(BattleEndReason::InternalError);
        return false;
    }
    if (!room_runtime_.complete_transition()) {
        end_battle_(BattleEndReason::InternalError);
        return false;
    }
    if (!enter_current_room_()) {
        end_battle_(BattleEndReason::InternalError);
        return false;
    }
    return true;
}

std::unique_ptr<battle::BattleInstance> battle::BattleInstance::create(BattleInstanceConfig config) {
    if (!valid_room_configuration(config)) {
        return nullptr;
    }
    auto instance = std::unique_ptr<BattleInstance>(new BattleInstance(std::move(config)));
    if (instance->initialization_failed_) {
        return nullptr;
    }
    if (!instance->enter_current_room_()) {
        return nullptr;
    }
    return instance;
}

void battle::BattleInstance::collect_combat_events_() {
    for (auto& event : world_.attack_events()) {
        const auto start_tick = server_tick_;
        const auto windup_ticks = duration_to_ticks_(event.windup_seconds);
        const auto active_ticks = duration_to_ticks_(event.active_seconds);
        const auto recovery_ticks = duration_to_ticks_(event.recovery_seconds);

        const auto active_start_tick = start_tick + windup_ticks;
        const auto active_end_tick = active_start_tick + active_ticks;
        const auto recovery_end_tick = active_end_tick + recovery_ticks;
        pending_battle_events_.emplace_back(PendingBattleEvent{
            .event = {
                .event_id = next_event_id_++,
                .payload = BattleAttackEvent{
                    .attacker = event.attacker,
                    .kind = event.kind,
                    .direction = event.direction,
                    .action_id = event.action_id,
                    .start_tick = start_tick,
                    .active_start_tick = active_start_tick,
                    .active_end_tick = active_end_tick,
                    .recovery_end_tick = recovery_end_tick,
                }
            },
            .expire_tick = server_tick_ + EventHistoryTicks
        });
    }
    for (auto& event : world_.death_events()) {
        pending_battle_events_.emplace_back(PendingBattleEvent{
            .event = BattleEvent{
                .event_id = next_event_id_++,
                .payload = BattleDeathEvent{
                    .victim = event.victim,
                    .killer = event.killer,
                    .kind = event.kind,
                    .position = event.position,
                    .direction = event.direction,
                    .monster_kind = event.monster_kind,
                },
            },
            .expire_tick = server_tick_ + EventHistoryTicks,
        });
    }
    // World 的事件仅服务于当前 tick；复制到带过期 tick 的队列后立即清空，
    // 使网络层可重复发送短期历史，而 ECS 不会重复处理同一攻击或死亡。
    world_.clear_attack_events();
    world_.clear_death_events();
}

void battle::BattleInstance::discard_expired_combat_events_() {
    std::erase_if(pending_battle_events_, [this](const PendingBattleEvent& event) {
        return event.expire_tick <= server_tick_;
    });
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
        // 只有玩家击杀会进入局内经验和 rcenter 结算统计；怪物或环境来源的事件
        // 没有 player 映射，因此直接忽略。
        const auto player_id = killer_it->second;
        grant_experience_(player_id, experience_for_monster_kind_(event.monster_kind));
        auto& stats = player_battle_stats_[player_id];
        stats.total_kills++;
        stats.kills_by_kind[event.monster_kind]++;
    }
    world_.clear_kill_events();
}

void battle::BattleInstance::tick_blessing_selection_(ecs::DeltaTime delta_time) {
    if (all_reward_choices_completed_()) {
        room_runtime_.begin_exit_selection();
        return;
    }
    reward_selection_.remaining_seconds -= delta_time;
    if (reward_selection_.remaining_seconds <= ecs::DeltaTime{0}) {
        apply_default_upgrade_choices_();
        room_runtime_.begin_exit_selection();
    }
}

void battle::BattleInstance::tick_fighting_(ecs::DeltaTime delta_time) {
    world_.tick(delta_time);
    collect_combat_events_();
    consume_kill_events_();
    // 先判失败，防止“最后一只怪物与最后一名玩家同 tick 死亡”被误结算为胜利。
    if (!world_.has_living_players()) {
        end_battle_(BattleEndReason::Defeat);
        return;
    }
    update_room_completion_();
}


void battle::BattleInstance::start_reward_selection_() {
    reward_selection_.remaining_seconds = SelectionTime;
    // 只有拥有待选次数的玩家获得候选。其余玩家显式清空旧选项，避免客户端快照
    // 在进入新一轮选择时继续显示过期按钮。
    for (auto& [player_id, blessing_state] : player_blessings_) {
        const auto progress = player_progress(player_id);
        if (!progress.has_value() || progress->pending_upgrade_choices <= 0) {
            blessing_state.current_options.clear();
            continue;
        }
        blessing_state.current_options = generate_blessing_options_(player_id);
    }
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


void battle::BattleInstance::grant_experience_(std::int64_t player_id, int experience) {
    if (experience <= 0) {
        return;
    }
    auto it = player_entities_.find(player_id);
    if (it == player_entities_.end()) {
        return;
    }
    ecs::Entity entity = it->second;
    auto progress = world_.registry().try_get<ecs::PlayerProgress>(entity);
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
    case MonsterKind::Ranged:
        return progression_config_.ranged_experience;
    }

    return 0;
}

int battle::BattleInstance::experience_to_next_level_(int level) const {
    return progression_config_.base_experience_to_next_level +
        (level - 1) * progression_config_.experience_to_next_level_growth;
}

std::vector<battle::BlessingOption> battle::BattleInstance::generate_blessing_options_(std::int64_t player_id) {
    auto candidates = AllBlessingIDs;
    std::ranges::shuffle(candidates, reward_random_engine_);

    std::vector<BlessingOption> options;
    options.reserve(RewardOptionCount);
    for (std::size_t i = 0; i < RewardOptionCount; ++i) {
        options.emplace_back(BlessingOption{
            .option_id = static_cast<int>(i),
            .blessing_id = candidates[i],
        });
    }
    return options;
}

void battle::BattleInstance::add_or_level_up_blessing_(ecs::Entity player_entity, PlayerBlessingState& blessing_state,
                                                       BlessingID blessing_id) {
    auto blessing_it = std::ranges::find_if(blessing_state.blessings, [blessing_id](const PlayerBlessing& blessing) {
        return blessing.blessing_id == blessing_id;
    });

    auto inventory = world_.registry().try_get<ecs::BlessingInventory>(player_entity);
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

bool battle::BattleInstance::all_reward_choices_completed_() const {
    return std::ranges::none_of(player_entities_, [this](const auto& p) {
        const auto* progress = world_.registry().try_get<ecs::PlayerProgress>(p.second);
        return progress && progress->pending_upgrade_choices > 0;
    });
}

std::uint64_t battle::BattleInstance::duration_to_ticks_(ecs::DeltaTime duration) const {
    if (duration <= ecs::DeltaTime{0}) {
        return 0;
    }
    constexpr float TickRoundingTolerance = 0.00001f;
    const auto scaled_ticks = duration.count() * static_cast<float>(tick_rate_);
    const auto nearest_tick = std::round(scaled_ticks);
    if (std::abs(scaled_ticks - nearest_tick) < TickRoundingTolerance) {
        return static_cast<std::uint64_t>(nearest_tick);
    }
    return static_cast<std::uint64_t>(std::ceil(scaled_ticks));
}

bool battle::BattleInstance::enter_current_room_() {
    const auto* current_room = dungeon_room_graph_.find_room(current_room_id());
    if (current_room == nullptr || !room_runtime_.prepare_current_room()) {
        return false;
    }

    std::vector<ecs::Entity> spawned_entities;
    auto revert = [this, &spawned_entities]() {
        for (const auto entity : spawned_entities) {
            world_.destroy_entity(entity);
        }
    };
    for (const auto& config : room_runtime_.monster_configs()) {
        if (auto entity = world_.create_monster(config); entity == ecs::NullEntity) {
            revert();
            return false;
        } else {
            spawned_entities.emplace_back(entity);
        }
    }
    if (!room_runtime_.start_current_room()) {
        revert();
        return false;
    }
    room_exit_choices_.clear();
    pending_battle_events_.emplace_back(PendingBattleEvent{
        .event = BattleEvent{
            .event_id = next_event_id_++,
            .payload = BattleRoomEnteredEvent{
                .room_id = current_room_id(),
                .layout_id = current_room->layout_id,
            },
        },
        .expire_tick = server_tick_ + EventHistoryTicks,
    });
    return true;
}

bool battle::BattleInstance::update_room_completion_() {
    if (!room_runtime_.update_living_monster_count(world_.living_monster_count())) {
        return false;
    }

    auto* current_room = dungeon_room_graph_.find_room(current_room_id());
    if (!current_room) {
        return false;
    }
    pending_battle_events_.emplace_back(PendingBattleEvent{
        .event = BattleEvent{
            .event_id = next_event_id_++,
            .payload = BattleRoomClearedEvent{
                .room_id = current_room_id(),
            },
        },
        .expire_tick = server_tick_ + EventHistoryTicks,
    });
    if (current_room->next_room_ids.empty()) {
        end_battle_(BattleEndReason::Victory);
        return true;
    }
    bool has_pending_choice = false;
    for (const auto player_id : player_entities_ | std::views::keys) {
        auto progress = player_progress(player_id);
        if (progress.has_value() && progress.value().pending_upgrade_choices > 0) {
            has_pending_choice = true;
            break;
        }
    }
    if (!has_pending_choice) {
        return room_runtime_.begin_exit_selection();
    }
    if (!room_runtime_.begin_blessing_selection()) {
        return false;
    }
    start_reward_selection_();
    return true;
}
