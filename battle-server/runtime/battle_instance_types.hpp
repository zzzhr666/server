#pragma once

#include <optional>
#include <string>
#include <unordered_map>
#include <vector>
#include <variant>

#include "ecs/world.hpp"
#include "gameplay/blessing.hpp"
#include "gameplay/growth.hpp"
#include "gameplay/monster_kind.hpp"
#include "gameplay/wave_planner.hpp"
#include "gameplay/weapon.hpp"

namespace battle {
    struct ProgressionConfig {
        int base_experience_to_next_level = 100;
        int experience_to_next_level_growth = 50;
        int melee_experience = 35;
        int ranged_experience = 45;
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

    struct BattleInstanceConfig {
        std::string room_name;
        std::vector<std::int64_t> player_ids;
        WaveConfig wave_config = default_wave_config();
        std::unordered_map<std::int64_t, std::pair<WeaponKind, GrowthLevels>> player_loadouts;
        std::optional<ecs::CreatePlayerConfig> player_config_override;
        std::optional<std::uint32_t> reward_random_seed;
        ecs::WorldBounds world_bounds = ecs::WorldBounds{
            .min_x = -20.0f,
            .max_x = 20.0f,
            .min_y = -20.0f,
            .max_y = 20.0f,
        };
        ProgressionConfig progression_config;
    };

    enum class BattleState : std::uint8_t {
        Running,
        Ended
    };

    enum class BattleEndReason : std::uint8_t {
        None,
        Victory,
        Defeat,
    };

    struct BattleAttackEvent {
        ecs::Entity attacker;
        ecs::AttackKind kind{};
        ecs::Direction direction{};
        ecs::CombatActionID action_id{};
    };

    struct BattleDeathEvent {
        ecs::Entity victim;
        ecs::Entity killer;
        ecs::DeathEntityKind kind{};
        ecs::Position position{};
        ecs::Direction direction{};
        std::optional<MonsterKind> monster_kind{};
    };

    struct BattleEvent {
        std::uint64_t event_id{};
        std::variant<BattleAttackEvent, BattleDeathEvent> payload;
    };

    struct BattleEntitySnapshot {
        ecs::Entity entity;
        ecs::EntityKind kind{};
        std::int64_t player_id{};
        ecs::Position position{};
        ecs::Direction direction{};
        int current_health{};
        int max_health{};
        std::optional<MonsterKind> monster_kind{};
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
    };

    enum class BattlePhase : std::uint8_t {
        Fighting,
        RewardSelection,
    };

    struct RewardSelectionState {
        ecs::DeltaTime remaining_seconds{0.0f};
    };

    constexpr ecs::DeltaTime SelectionTime{15.0f};

    struct PlayerProgressSnapshot {
        std::int64_t player_id = 0;
        int level = 1;
        int experience = 0;
        int experience_to_next_level = 0;
        int pending_upgrade_choices = 0;
    };

    struct BattleWorldSnapshot {
        std::vector<BattleEntitySnapshot> entities;
        std::size_t current_wave = 0;
        BattlePhase phase = BattlePhase::Fighting;
        ecs::DeltaTime reward_selection_remaining{0.0f};
        std::vector<PlayerProgressSnapshot> player_progress;
        std::vector<PlayerBlessingState> player_blessings;
        std::uint64_t server_tick{};
        std::vector<BattleEvent> events;
    };
}
