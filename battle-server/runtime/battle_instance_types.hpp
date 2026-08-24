#pragma once

#include <optional>
#include <string>
#include <unordered_map>
#include <vector>
#include <variant>

#include "ecs/world.hpp"
#include "gameplay/blessing.hpp"
#include "gameplay/gameplay_config.hpp"
#include "gameplay/growth.hpp"
#include "gameplay/monster_kind.hpp"
#include "gameplay/hero.hpp"
#include "gameplay/room_flow.hpp"
#include "gameplay/room_graph.hpp"
#include "gameplay/room_graph_presets.hpp"
#include "gameplay/room_layout_catalog.hpp"
#include "gameplay/shop_item.hpp"
#include "gameplay/shop_item_catalog.hpp"

namespace battle {
    struct ProgressionConfig {
        int base_experience_to_next_level = gameplay_config::progression::BaseExperienceToNextLevel;
        int experience_to_next_level_growth = gameplay_config::progression::ExperienceToNextLevelGrowth;
        int melee_experience = gameplay_config::progression::MeleeMonsterExperience;
        int ranged_experience = gameplay_config::progression::RangedMonsterExperience;
    };

    struct BlessingOption {
        int option_id = 0;
        BlessingID blessing_id = BlessingID::BurnOnHit;
    };

    struct PlayerBlessingState {
        std::int64_t player_id = 0;
        std::vector<PlayerBlessing> blessings;
        std::vector<BlessingOption> current_options;
    };

    enum class FreeRewardKind : std::uint8_t {
        Heal,
        Attack,
        DamageReduction,
        Blessing,
        Skip,
    };

    struct PlayerFreeRewardState {
        std::int64_t player_id{};
        bool completed = false;
        std::optional<FreeRewardKind> selected_kind;
    };

    struct PlayerSoulSnapshot {
        std::int64_t player_id{};
        int souls{};
    };

    constexpr std::uint32_t DefaultBattleTickRate = 60;

    struct BattleInstanceConfig {
        std::string room_name;
        std::vector<std::int64_t> player_ids;
        std::unordered_map<std::int64_t, std::pair<HeroKind, GrowthLevels>> player_loadouts;
        std::optional<ecs::CreatePlayerConfig> player_config_override;
        std::optional<std::uint32_t> reward_random_seed;
        std::vector<ShopOffer> shop_offers = default_shop_offers();
        std::vector<ShopItemDefinition> shop_item_definitions = default_shop_item_definitions();
        ecs::WorldBounds world_bounds = ecs::WorldBounds{
            .min_x = gameplay_config::room::MinCoordinate,
            .max_x = gameplay_config::room::MaxCoordinate,
            .min_y = gameplay_config::room::MinCoordinate,
            .max_y = gameplay_config::room::MaxCoordinate,
        };
        ProgressionConfig progression_config;
        std::uint32_t tick_rate = DefaultBattleTickRate;
        DungeonRoomGraph dungeon_room_graph = default_dungeon_room_graph();
        RoomLayoutCatalog room_layout_catalog = default_room_layout_catalog();
    };

    enum class BattleState : std::uint8_t {
        Running,
        Ended
    };

    enum class BattleEndReason : std::uint8_t {
        None,
        Victory,
        Defeat,
        InternalError,
    };

    struct BattleAttackEvent {
        ecs::Entity attacker;
        ecs::AttackKind kind{};
        ecs::Direction direction{};
        ecs::CombatActionID action_id{};
        // 四个边界组成半开区间，依次表示 Windup、Active 和 Recovery。
        std::uint64_t start_tick{};
        std::uint64_t active_start_tick{};
        std::uint64_t active_end_tick{};
        std::uint64_t recovery_end_tick{};
    };

    struct BattleDeathEvent {
        ecs::Entity victim;
        ecs::Entity killer;
        ecs::DeathEntityKind kind{};
        ecs::Position position{};
        ecs::Direction direction{};
        std::optional<MonsterKind> monster_kind{};
    };

    struct BattleRoomClearedEvent {
        DungeonRoomID room_id = 0;
    };

    struct BattleRoomEnteredEvent {
        DungeonRoomID room_id = 0;
        std::string layout_id;
    };

    struct BattleEvent {
        std::uint64_t event_id{};
        std::variant<BattleAttackEvent, BattleDeathEvent, BattleRoomClearedEvent, BattleRoomEnteredEvent> payload;
    };

    struct BattleEntitySnapshot {
        ecs::Entity entity;
        ecs::EntityKind kind{};
        std::int64_t player_id{};
        std::optional<HeroKind> hero;
        ecs::Position position{};
        ecs::Direction direction{};
        int current_health{};
        int max_health{};
        std::optional<MonsterKind> monster_kind{};
        float collision_radius{};
        std::string scene_object_kind;
    };

    struct PlayerBattleStats {
        int total_kills = 0;
        std::unordered_map<MonsterKind, int> kills_by_kind;
    };

    struct MonsterKillCount {
        MonsterKind monster_kind = MonsterKind::Melee;
        int count = 0;
    };

    struct PlayerSettlement {
        std::int64_t player_id = 0;
        int total_kills = 0;
        std::vector<MonsterKillCount> kills;
    };

    struct BattleSettlement {
        std::vector<PlayerSettlement> players;
        BattleEndReason reason = BattleEndReason::None;
        std::int64_t combat_duration_ms = 0;
    };

    struct RewardSelectionState {
        ecs::DeltaTime remaining_seconds{0.0f};
    };

    struct PlayerProgressSnapshot {
        std::int64_t player_id = 0;
        int level = 1;
        int experience = 0;
        int experience_to_next_level = 0;
        int pending_upgrade_choices = 0;
    };

    struct PlayerCombatStatsSnapshot {
        std::int64_t player_id = 0;
        int attack_damage = 0;
        float move_speed = 0.0f;
        float attack_cooldown_seconds = 0.0f;
        int armor = 0;
    };

    struct PlayerRoomExitChoiceSnapshot {
        std::int64_t player_id = 0;
        DungeonRoomID room_exit_id = 0;
    };

    struct BattleWorldSnapshot {
        std::vector<BattleEntitySnapshot> entities;
        ecs::DeltaTime reward_selection_remaining{0.0f};
        std::vector<PlayerProgressSnapshot> player_progress;
        std::vector<PlayerCombatStatsSnapshot> player_combat_stats;
        std::vector<PlayerBlessingState> player_blessings;
        std::uint64_t server_tick{};
        std::vector<BattleEvent> events;
        std::uint32_t tick_rate = DefaultBattleTickRate;
        DungeonRoomID current_room_id = 0;
        RoomFlowState room_state = RoomFlowState::EnteringRoom;
        std::vector<DungeonRoomID> available_room_exit_ids;
        std::vector<PlayerRoomExitChoiceSnapshot> player_room_exit_choices;
        std::string current_room_layout_id;
        std::vector<PlayerFreeRewardState> free_reward_states;
        std::vector<ShopOffer> shop_offers;
        std::vector<PlayerSoulSnapshot> player_souls;
        std::vector<ShopItemDefinition> shop_item_definitions;
        std::unordered_map<std::int64_t, std::vector<std::uint32_t>> purchased_shop_items;
    };
}
