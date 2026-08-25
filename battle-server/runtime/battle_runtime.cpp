#include "battle_runtime.hpp"

#include <utility>
#include <vector>
#include <chrono>
#include <type_traits>
#include <variant>

#include "battle_instance.hpp"
#include "game/game_manager.hpp"
#include "gameplay/growth.hpp"
#include "gameplay/monster_kind_codec.hpp"
#include "gameplay/hero.hpp"
#include "net/packet_codec.hpp"
#include "platform/metrics.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"
#include "spdlog/spdlog.h"

namespace {
    battle::v1::EntityKind to_proto_entity_kind(battle::ecs::EntityKind entity_kind) {
        switch (entity_kind) {
        case battle::ecs::EntityKind::Player:
            return battle::v1::ENTITY_KIND_PLAYER;
        case battle::ecs::EntityKind::Monster:
            return battle::v1::ENTITY_KIND_MONSTER;
        case battle::ecs::EntityKind::Projectile:
            return battle::v1::ENTITY_KIND_PROJECTILE;
        case battle::ecs::EntityKind::Obstacle:
            return battle::v1::ENTITY_KIND_OBSTACLE;
        case battle::ecs::EntityKind::Trap:
            return battle::v1::ENTITY_KIND_TRAP;
        default:
            return battle::v1::ENTITY_KIND_UNSPECIFIED;
        }
    }

    std::string boss_phase_to_string(battle::ecs::BossPhase phase) {
        switch (phase) {
        case battle::ecs::BossPhase::One:
            return "one";
        case battle::ecs::BossPhase::Two:
            return "two";
        }
        return "unknown";
    }

    std::string boss_ability_kind_to_string(battle::ecs::BossAbilityKind kind) {
        switch (kind) {
        case battle::ecs::BossAbilityKind::None:
            return "none";
        case battle::ecs::BossAbilityKind::TripleDash:
            return "triple_dash";
        case battle::ecs::BossAbilityKind::RadialProjectile:
            return "radial_projectile";
        case battle::ecs::BossAbilityKind::Tornado:
            return "tornado";
        }
        return "unknown";
    }

    std::string attack_phase_to_string(battle::ecs::AttackPhase phase) {
        switch (phase) {
        case battle::ecs::AttackPhase::Idle:
            return "idle";
        case battle::ecs::AttackPhase::Windup:
            return "windup";
        case battle::ecs::AttackPhase::Active:
            return "active";
        case battle::ecs::AttackPhase::Recovery:
            return "recovery";
        }
        return "unknown";
    }


    std::string battle_end_reason_to_string(battle::BattleEndReason reason) {
        switch (reason) {
        case battle::BattleEndReason::Defeat: {
            return "defeat";
        }
        case battle::BattleEndReason::Victory: {
            return "victory";
        }
        case battle::BattleEndReason::InternalError: {
            return "internal_error";
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

    battle::v1::RoomFlowState to_proto_room_flow_state(battle::RoomFlowState state) {
        switch (state) {
        case battle::RoomFlowState::EnteringRoom: {
            return battle::v1::ROOM_FLOW_STATE_ENTERING_ROOM;
        }
        case battle::RoomFlowState::Fighting: {
            return battle::v1::ROOM_FLOW_STATE_FIGHTING;
        }
        case battle::RoomFlowState::RoomCleared: {
            return battle::v1::ROOM_FLOW_STATE_ROOM_CLEARED;
        }
        case battle::RoomFlowState::ChoosingBlessing: {
            return battle::v1::ROOM_FLOW_STATE_CHOOSING_BLESSING;
        }
        case battle::RoomFlowState::ChoosingExit: {
            return battle::v1::ROOM_FLOW_STATE_CHOOSING_EXIT;
        }
        case battle::RoomFlowState::Transitioning: {
            return battle::v1::ROOM_FLOW_STATE_TRANSITIONING;
        }
        case battle::RoomFlowState::Rewarding: {
            return battle::v1::ROOM_FLOW_STATE_REWARDING;
        }
        default: {
            return battle::v1::ROOM_FLOW_STATE_UNSPECIFIED;
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

    battle::v1::FreeRewardKind to_proto_free_reward_kind(battle::FreeRewardKind kind) {
        switch (kind) {
        case battle::FreeRewardKind::Heal:
            return battle::v1::FREE_REWARD_KIND_HEAL;
        case battle::FreeRewardKind::Attack:
            return battle::v1::FREE_REWARD_KIND_ATTACK;
        case battle::FreeRewardKind::DamageReduction:
            return battle::v1::FREE_REWARD_KIND_DAMAGE_REDUCTION;
        case battle::FreeRewardKind::Blessing:
            return battle::v1::FREE_REWARD_KIND_BLESSING;
        case battle::FreeRewardKind::Skip:
            return battle::v1::FREE_REWARD_KIND_SKIP;
        }
        return battle::v1::FREE_REWARD_KIND_UNSPECIFIED;
    }

    battle::v1::ShopBuffKind to_proto_shop_buff_kind(battle::ShopBuffKind kind) {
        switch (kind) {
        case battle::ShopBuffKind::AttackDamage:
            return battle::v1::SHOP_BUFF_KIND_ATTACK_DAMAGE;
        case battle::ShopBuffKind::MaxHealth:
            return battle::v1::SHOP_BUFF_KIND_MAX_HEALTH;
        case battle::ShopBuffKind::Armor:
            return battle::v1::SHOP_BUFF_KIND_ARMOR;
        case battle::ShopBuffKind::MoveSpeed:
            return battle::v1::SHOP_BUFF_KIND_MOVE_SPEED;
        }
        return battle::v1::SHOP_BUFF_KIND_UNSPECIFIED;
    }

    battle::v1::AttackKind to_proto_attack_kind(battle::ecs::AttackKind kind) {
        switch (kind) {
        case battle::ecs::AttackKind::Melee:
            return battle::v1::ATTACK_KIND_MELEE;
        case battle::ecs::AttackKind::Projectile:
            return battle::v1::ATTACK_KIND_PROJECTILE;
        }
        return battle::v1::ATTACK_KIND_UNSPECIFIED;
    }

    battle::v1::EntityKind to_proto_death_entity_kind(battle::ecs::DeathEntityKind kind) {
        switch (kind) {
        case battle::ecs::DeathEntityKind::Player:
            return battle::v1::ENTITY_KIND_PLAYER;
        case battle::ecs::DeathEntityKind::Monster:
            return battle::v1::ENTITY_KIND_MONSTER;
        }
        return battle::v1::ENTITY_KIND_UNSPECIFIED;
    }


    battle::v1::ServerPacket make_snapshot(const std::string& room_name, const battle::BattleWorldSnapshot& snapshot) {
        battle::v1::ServerPacket packet;
        auto send_pkg = packet.mutable_snapshot();
        send_pkg->set_room_name(room_name);
        send_pkg->set_reward_selection_remaining_seconds(snapshot.reward_selection_remaining.count());
        send_pkg->set_server_tick(snapshot.server_tick);
        send_pkg->set_tick_rate(snapshot.tick_rate);
        send_pkg->set_current_room_id(snapshot.current_room_id);
        send_pkg->set_room_state(to_proto_room_flow_state(snapshot.room_state));
        send_pkg->set_current_room_layout_id(snapshot.current_room_layout_id);
        for (const auto exit_id : snapshot.available_room_exit_ids) {
            send_pkg->add_available_room_exit_ids(exit_id);
        }
        for (const auto& entity : snapshot.entities) {
            auto entity_snapshot = send_pkg->add_entities();
            entity_snapshot->set_entity(entity.entity.packed());
            entity_snapshot->set_kind(to_proto_entity_kind(entity.kind));
            entity_snapshot->set_player_id(entity.player_id);
            entity_snapshot->mutable_position()->set_x(entity.position.x);
            entity_snapshot->mutable_position()->set_y(entity.position.y);
            entity_snapshot->mutable_direction()->set_x(entity.direction.x);
            entity_snapshot->mutable_direction()->set_y(entity.direction.y);
            entity_snapshot->set_current_health(entity.current_health);
            entity_snapshot->set_max_health(entity.max_health);
            entity_snapshot->set_collision_radius(entity.collision_radius);
            entity_snapshot->set_scene_object_kind(entity.scene_object_kind);

            if (entity.hero.has_value()) {
                entity_snapshot->set_hero(battle::hero_kind_to_string(entity.hero.value()));
            }
            if (entity.monster_kind.has_value()) {
                entity_snapshot->set_monster_kind(battle::monster_kind_to_string(entity.monster_kind.value()));
            }
            if (entity.boss_phase.has_value()) {
                entity_snapshot->set_boss_phase(boss_phase_to_string(entity.boss_phase.value()));
            }
            if (entity.boss_ability.has_value()) {
                entity_snapshot->set_boss_ability(boss_ability_kind_to_string(entity.boss_ability.value()));
            }
            if (entity.boss_action_phase.has_value()) {
                entity_snapshot->set_boss_action_phase(attack_phase_to_string(entity.boss_action_phase.value()));
            }
            entity_snapshot->set_boss_ability_remaining_seconds(entity.boss_ability_remaining_seconds);
            entity_snapshot->set_boss_sequence_index(entity.boss_sequence_index);
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
        for (const auto& battle_event : snapshot.events) {
            auto event = send_pkg->add_events();
            event->set_event_id(battle_event.event_id);
            std::visit([event]<typename T>(const T& payload) {
                using Payload = std::decay_t<T>;
                if constexpr (std::is_same_v<Payload, battle::BattleAttackEvent>) {
                    auto proto_attack = event->mutable_attack();
                    proto_attack->set_attacker_entity(payload.attacker.packed());
                    proto_attack->set_action_id(payload.action_id);
                    proto_attack->set_attack_kind(to_proto_attack_kind(payload.kind));
                    proto_attack->mutable_direction()->set_x(payload.direction.x);
                    proto_attack->mutable_direction()->set_y(payload.direction.y);
                    proto_attack->set_start_tick(payload.start_tick);
                    proto_attack->set_active_start_tick(payload.active_start_tick);
                    proto_attack->set_active_end_tick(payload.active_end_tick);
                    proto_attack->set_recovery_end_tick(payload.recovery_end_tick);
                } else if constexpr (std::is_same_v<Payload, battle::BattleDeathEvent>) {
                    auto proto_death = event->mutable_death();
                    proto_death->set_victim_entity(payload.victim.packed());
                    proto_death->set_killer_entity(payload.killer.packed());
                    proto_death->set_victim_kind(to_proto_death_entity_kind(payload.kind));
                    proto_death->mutable_direction()->set_x(payload.direction.x);
                    proto_death->mutable_direction()->set_y(payload.direction.y);
                    proto_death->mutable_position()->set_x(payload.position.x);
                    proto_death->mutable_position()->set_y(payload.position.y);
                    if (payload.monster_kind.has_value()) {
                        proto_death->set_monster_kind(battle::monster_kind_to_string(payload.monster_kind.value()));
                    }
                } else if constexpr (std::is_same_v<Payload, battle::BattleRoomClearedEvent>) {
                    auto proto_cleared = event->mutable_room_cleared();
                    proto_cleared->set_room_id(payload.room_id);
                } else if constexpr (std::is_same_v<Payload, battle::BattleRoomEnteredEvent>) {
                    auto proto_entered = event->mutable_room_entered();
                    proto_entered->set_room_id(payload.room_id);
                    proto_entered->set_layout_id(payload.layout_id);
                }
            }, battle_event.payload);
        }
        for (const auto& choice : snapshot.player_room_exit_choices) {
            auto* proto_choice = send_pkg->add_player_room_exit_choices();
            proto_choice->set_player_id(choice.player_id);
            proto_choice->set_room_exit_id(choice.room_exit_id);
        }
        for (const auto& free_reward : snapshot.free_reward_states) {
            auto* proto_free_reward = send_pkg->add_free_reward_states();
            proto_free_reward->set_player_id(free_reward.player_id);
            proto_free_reward->set_completed(free_reward.completed);
            if (free_reward.selected_kind.has_value()) {
                proto_free_reward->set_selected_kind(to_proto_free_reward_kind(free_reward.selected_kind.value()));
            }
        }
        for (const auto& offer : snapshot.shop_offers) {
            auto* proto_offer = send_pkg->add_shop_offers();
            proto_offer->set_item_id(offer.item_id);
            proto_offer->set_price(offer.price);
        }
        for (const auto& player_soul : snapshot.player_souls) {
            auto* proto_player_soul = send_pkg->add_player_souls();
            proto_player_soul->set_player_id(player_soul.player_id);
            proto_player_soul->set_souls(player_soul.souls);
        }
        for (const auto& stats : snapshot.player_combat_stats) {
            auto* proto_stats = send_pkg->add_player_combat_stats();
            proto_stats->set_player_id(stats.player_id);
            proto_stats->set_attack_damage(stats.attack_damage);
            proto_stats->set_move_speed(stats.move_speed);
            proto_stats->set_attack_cooldown_seconds(stats.attack_cooldown_seconds);
            proto_stats->set_armor(stats.armor);
        }
        for (const auto& definition : snapshot.shop_item_definitions) {
            auto* proto_definition = send_pkg->add_shop_item_definitions();
            proto_definition->set_item_id(definition.item_id);
            proto_definition->set_item_name(definition.item_name);
            for (const auto& buff : definition.buffs) {
                auto* proto_buff = proto_definition->add_buffs();
                proto_buff->set_kind(to_proto_shop_buff_kind(buff.kind));
                proto_buff->set_value(buff.value);
            }
        }
        for (const auto& [player_id, item_ids] : snapshot.purchased_shop_items) {
            auto* proto_purchased_items = send_pkg->add_purchased_shop_items();
            proto_purchased_items->set_player_id(player_id);
            for (const auto item_id : item_ids) {
                proto_purchased_items->add_item_ids(item_id);
            }
        }
        return packet;
    }

    std::chrono::steady_clock::duration tick_interval_from_rate(std::uint32_t tick_rate) {
        const auto effective_tick_rate = tick_rate > 0 ? tick_rate : battle::DefaultBattleTickRate;
        return std::chrono::duration_cast<std::chrono::steady_clock::duration>(
            std::chrono::duration<double>(1.0 / static_cast<double>(effective_tick_rate)));
    }

    battle::ecs::DeltaTime fixed_delta_time_from_rate(std::uint32_t tick_rate) {
        const auto effective_tick_rate = tick_rate > 0 ? tick_rate : battle::DefaultBattleTickRate;
        return battle::ecs::DeltaTime{1.0f / static_cast<float>(effective_tick_rate)};
    }

    std::uint32_t normalize_tick_rate(int tick_rate) {
        if (tick_rate <= 0) {
            return battle::DefaultBattleTickRate;
        }
        return static_cast<std::uint32_t>(tick_rate);
    }
}

battle::BattleRuntime::BattleRuntime(RoomManager& room_manager, SessionManager& session_manager,
                                     BattleMetrics& metrics,
                                     SendPacketCallback send_packet_callback, BattleInstanceFactory factory,
                                     FinishMatchCallback finish_match_callback, int tick_rate,
                                     std::chrono::seconds session_idle_timeout_seconds,
                                     std::chrono::seconds all_players_disconnected_timeout_seconds)
    : room_manager_(room_manager), session_manager_(session_manager), metrics_(metrics),
      send_packet_(std::move(send_packet_callback)), finish_match_callback_(std::move(finish_match_callback)),
      running_(false), instance_factory_(std::move(factory)), tick_rate_(normalize_tick_rate(tick_rate)),
      fixed_delta_time_(fixed_delta_time_from_rate(tick_rate_)), tick_interval_(tick_interval_from_rate(tick_rate_)),
      session_idle_timeout_(session_idle_timeout_seconds),
      all_players_disconnected_timeout_(all_players_disconnected_timeout_seconds) {
    if (!instance_factory_) {
        instance_factory_ = [](BattleInstanceConfig config) {
            return BattleInstance::create(std::move(config));
        };
    }
}

battle::BattleRuntime::~BattleRuntime() {
    stop();
}

void battle::BattleRuntime::start_room(const std::string& room_name) {
    {
        std::lock_guard<std::mutex> lock(mutex_);
        // all_players_joined 的 hello 可能并发到达。starting_rooms_ 覆盖实例创建
        // 期间的空窗，instances_ 覆盖创建完成后，二者共同保证每个房间只有一个 World。
        if (starting_rooms_.contains(room_name) || instances_.contains(room_name)) {
            SPDLOG_DEBUG("room start ignored room={} reason=already_started", room_name);
            return;
        }
        starting_rooms_.insert(room_name);
    }

    // 在不持有 runtime 锁时读取房间配置和 session，避免锁住 tick 线程；房间启动
    // 只依赖完整 roster，重连期间连接状态变化不会改变 BattleInstance 的玩家集合。
    auto sessions = session_manager_.sessions_in_room(room_name);
    auto configured_loadouts = room_manager_.player_loadouts(room_name);

    std::vector<std::int64_t> player_ids;
    player_ids.reserve(sessions.size());
    for (const auto& session : sessions) {
        player_ids.push_back(session->player_id());
    }
    std::unordered_map<std::int64_t, std::pair<HeroKind, GrowthLevels>> player_loadouts;
    player_loadouts.reserve(configured_loadouts.size());
    // 英雄文本无效时保留 BattleInstance 的默认 Fire 配置，避免一个脏 loadout 阻止整个
    // 已匹配房间开始；局外成长在此冻结，战斗中不再读取 Redis 或 rcenter。
    for (const auto& loadout : configured_loadouts) {
        auto hero_kind = hero_kind_from_string(loadout.hero);
        if (!hero_kind.has_value()) {
            continue;
        }
        auto growth_level = GrowthLevels{
            .attack_level = loadout.attack_level,
            .attack_speed_level = loadout.attack_speed_level,
            .health_level = loadout.health_level,
            .move_speed_level = loadout.move_speed_level,
        };
        player_loadouts.emplace(loadout.player_id, std::make_pair(hero_kind.value(), growth_level));
    }

    auto instance = instance_factory_(BattleInstanceConfig{
        .room_name = room_name,
        .player_ids = player_ids,
        .player_loadouts = std::move(player_loadouts),
        .tick_rate = tick_rate_,
    });

    if (!instance) {
        std::lock_guard<std::mutex> lock(mutex_);
        starting_rooms_.erase(room_name);
        SPDLOG_ERROR("battle instance creation failed room={}", room_name);
        return;
    }

    SPDLOG_INFO("battle instance started room={} players={}", room_name, player_ids.size());

    auto game_start_packet = make_game_start(room_name, player_ids);

    {
        std::lock_guard<std::mutex> lock(mutex_);
        starting_rooms_.erase(room_name);
        instances_.emplace(room_name, std::move(instance));
    }

    // 只通知当前连接的端点。断线玩家之后 hello 重绑后仍能从普通快照继续同步，
    // 无需为其保留或重放 GameStart 包。
    for (const auto& session : session_manager_.connected_sessions_in_room(room_name)) {
        send_packet_(game_start_packet, session->endpoint());
    }
}

void battle::BattleRuntime::tick(ecs::DeltaTime delta_time) {
    const auto tick_started_at = std::chrono::steady_clock::now();
    std::vector<v1::ServerPacket> packets;
    std::vector<UdpEndpoint> endpoints;
    std::vector<std::pair<std::string, BattleEndReason>> end_state;
    const auto now = std::chrono::steady_clock::now();
    // 会话超时先于房间检查执行。这样同一 tick 内，刚超过阈值的最后一个玩家会
    // 立即进入“全员断线”计时，而不会多保留一个 tick 的伪连接状态。
    session_manager_.mark_stale_sessions(now, session_idle_timeout_);
    std::vector<std::string> empty_rooms;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        for (auto& [room_name, instance] : instances_) {
            auto connected_sessions = session_manager_.connected_sessions_in_room(room_name);
            std::unordered_set<std::int64_t> online_players;
            for (const auto& session : connected_sessions) {
                online_players.insert(session->player_id());
            }
            instance->update_connected_players(online_players);
            instance->tick(delta_time);
            auto snapshot = instance->snapshot();
            const auto packet = make_snapshot(room_name, snapshot);
            for (const auto& session : connected_sessions) {
                packets.emplace_back(packet);
                endpoints.push_back(session->endpoint());
            }

            if (instance->ended()) {
                auto reason = instance->end_reason();
                end_state.emplace_back(room_name, reason);
                continue;
            }

            if (!connected_sessions.empty()) {
                // 任何玩家成功重连都会取消无人房间倒计时；房间本身和 ECS 世界不重建。
                all_disconnected_since_.erase(room_name);
                continue;
            }

            // 首次全员断线只记录开始时间，不立即结束。超时期间客户端仍可通过 hello
            // 重绑 session；达到阈值后统一走 end_room，保证资源释放与 FinishMatch 一致。
            const auto [it, inserted] = all_disconnected_since_.try_emplace(room_name, now);
            if (!inserted && now - it->second >= all_players_disconnected_timeout_) {
                empty_rooms.emplace_back(room_name);
            }
        }
    }
    // 网络发送放在 runtime 锁外，避免慢 UDP send 阻塞战斗推进、输入和结束清理。
    for (std::size_t i = 0; i < packets.size(); i++) {
        send_packet_(packets[i], endpoints[i]);
    }

    for (const auto& [room_name,reason] : end_state) {
        end_room(room_name, battle_end_reason_to_string(reason));
    }
    for (const auto& room_name : empty_rooms) {
        end_room(room_name, "all_players_disconnected");
    }

    const auto elapsed = std::chrono::steady_clock::now() - tick_started_at;
    const auto elapsed_seconds = std::chrono::duration<double>(elapsed).count();
    metrics_.observe_tick_duration(elapsed_seconds);
    if (elapsed > tick_interval_) {
        metrics_.increment_tick_overrun();
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

bool battle::BattleRuntime::choose_free_reward(const std::string& room_name, std::int64_t player_id,
                                               FreeRewardKind kind) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = instances_.find(room_name);
    return it == instances_.end() ? false : it->second->choose_free_reward(player_id, kind);
}

bool battle::BattleRuntime::purchase_shop_item(const std::string& room_name, std::int64_t player_id,
                                               std::uint32_t item_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = instances_.find(room_name);
    return it == instances_.end() ? false : it->second->purchase_shop_item(player_id, item_id);
}

bool battle::BattleRuntime::select_room_exit(const std::string& room_name, std::int64_t player_id,
                                             DungeonRoomID next_room_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = instances_.find(room_name);
    return it == instances_.end() ? false : it->second->select_room_exit(player_id, next_room_id);
}

void battle::BattleRuntime::start() {
    bool expected = false;
    if (!running_.compare_exchange_strong(expected, true)) {
        SPDLOG_WARN("battle runtime start ignored reason=already_running");
        return;
    }
    SPDLOG_INFO("battle runtime started");
    tick_thread_ = std::thread([this]() {
        using clock = std::chrono::steady_clock;
        auto next_tick = clock::now();
        while (running_) {
            tick(fixed_delta_time_);

            // 墙上时间只负责调度，不参与模拟。使用绝对下一帧时间减少累积漂移；
            // 若单帧明显落后，则重置调度基准，避免持续追赶历史 tick 导致 CPU 空转。
            next_tick += tick_interval_;
            std::this_thread::sleep_until(next_tick);
            if (clock::now() > next_tick + tick_interval_) {
                next_tick = clock::now();
            }
        }
    });
}

void battle::BattleRuntime::stop() {
    running_ = false;
    if (tick_thread_.joinable()) {
        tick_thread_.join();
    }
    SPDLOG_INFO("battle runtime stopped");
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
            SPDLOG_WARN("room end requested for missing room={}", room_name);
            return {
                .status = EndRoomStatus::RoomNotFound,
                .message = "unable to find instance",
            };
        }
        // 先取结算和完整 session roster。即使部分玩家已断线，也必须将其 ID 传给
        // FinishMatch，才能清除 rcenter 的 ActiveMatch 和 inGame 标记。
        settlement = it->second->settlement();
        auto sessions = session_manager_.sessions_in_room(room_name);
        auto connected_sessions = session_manager_.connected_sessions_in_room(room_name);

        player_ids.reserve(sessions.size());
        for (const auto& session : sessions) {
            player_ids.push_back(session->player_id());
        }
        for (const auto& session : connected_sessions) {
            endpoints.emplace_back(session->endpoint());
        }
        packet = make_game_over(room_name, player_ids, reason, to_packet_player_stats(settlement));
        // 从实例表和无人倒计时移除后，后续输入立即查不到房间；实际 session 与 Room
        // 的清理放到锁外，避免跨管理器调用时形成锁顺序问题。
        instances_.erase(it);
        all_disconnected_since_.erase(room_name);
    }
    for (auto& endpoint : endpoints) {
        send_packet_(packet, endpoint);
    }

    // GameOver 仅发给仍连接的端点，但 session/room 的资源释放覆盖完整 roster。
    session_manager_.remove_room(room_name);
    room_manager_.close_room(room_name);
    SPDLOG_INFO("battle room resources released room={} reason={} players={}", room_name, reason, player_ids.size());
    FinishedBattle finished_battle{
        .room_name = room_name,
        .player_ids = player_ids,
        .settlement = settlement,
        .reason = reason,
    };
    if (finish_match_callback_) {
        finish_match_callback_(finished_battle);
    }
    return {
        .status = EndRoomStatus::OK,
        .message = "room ended",
    };
}
