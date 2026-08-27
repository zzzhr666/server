#pragma once

#include <cstdint>
#include <optional>
#include <vector>

#include "ecs/entity/entity.hpp"
#include "ecs/time.hpp"
#include "ecs/combat/combat.hpp"
#include "gameplay/blessing.hpp"
#include "gameplay/gameplay_config.hpp"
#include "gameplay/monster_kind.hpp"
#include "gameplay/trap_kind.hpp"

namespace battle::ecs {
    /// @brief 一帧网络输入转换后的玩家动作请求。
    struct PlayerCommand {
        float move_x;
        float move_y;
        bool attack_requested;
        bool dash_requested;
    };

    /// @brief 二维坐标或单位方向向量。
    struct Position {
        float x;
        float y;
    };

    /// @brief 实体当前面向或攻击方向。
    struct Direction {
        float x;
        float y;
    };

    /// @brief 实体的位置与朝向组合。
    struct Transform {
        Position position;
        Direction direction;
    };

    /// @brief 实体在本 tick 使用的二维速度。
    struct Velocity {
        float x;
        float y;
    };

    /// @brief 原始移动请求，由客户端或怪物 AI 写入。
    struct MoveRequest {
        float x;
        float y;
    };

    /// @brief 原始攻击请求。
    struct AttackRequest {
        bool requested;
    };

    /// @brief 原始冲刺请求。
    struct DashRequest {
        bool requested;
    };

    /// @brief 归一化后可由移动系统消费的移动意图。
    struct MoveIntent {
        float x;
        float y;
    };

    /// @brief 标记实体为玩家，由玩法系统据此区分阵营。
    struct PlayerController {};

    /// @brief 标记实体为怪物，由玩法系统据此区分阵营。
    struct MonsterController {};

    /// @brief 角色的基础移动属性。
    struct CharacterStats {
        float move_speed;
        int armor;
    };

    /// @brief 实体当前和最大生命值。
    struct Health {
        int current_health;
        int max_health;
    };

    /// @brief 攻击的命中方式。
    enum class AttackKind {
        Melee,
        Projectile,
    };

    /// @brief 当前 tick 解析出的冲刺意图。
    struct DashIntent {
        bool active;
        float dash_speed_multiplier;
    };

    /// @brief 攻击动作当前所处的权威时序阶段。
    enum class AttackPhase {
        Idle,
        Windup,
        Active,
        Recovery,
    };

    /// @brief 跨 tick 保留的攻击动作状态。
    struct AttackState {
        AttackPhase phase{AttackPhase::Idle};
        DeltaTime phase_remaining{};
        CombatContext context{};
        Direction locked_direction{};
        std::vector<Entity> hit_targets;
        bool projectile_spawned{false};
    };

    /// @brief 英雄或怪物的基础攻击定义。
    struct AttackDefinition {
        AttackKind kind;
        int damage;
        float range;
        DeltaTime cooldown_seconds;
        DeltaTime windup_seconds{};
        DeltaTime active_seconds{gameplay_config::combat::DefaultAttackActiveSeconds};
        DeltaTime recovery_seconds{};
        float movement_multiplier{1.0f};
        float projectile_speed;
        float projectile_hit_radius{gameplay_config::combat::DefaultProjectileHitRadius};
    };

    /// @brief 冲刺的冷却和速度倍率配置。
    struct Dash {
        DeltaTime cooldown_seconds;
        float dash_speed_multiplier;
    };

    /// @brief 攻击距离下一次可用的剩余时间。
    struct AttackCooldown {
        DeltaTime remaining_seconds;
    };

    /// @brief 冲刺距离下一次可用的剩余时间。
    struct DashCooldown {
        DeltaTime remaining_seconds;
    };

    /// @brief 伤害事件的来源类别，用于区分祝福触发规则。
    enum class DamageSourceKind {
        Attack,
        Burn,
        ChainLightning,
        Trap,
        Freeze,
    };

    /// @brief 在伤害修正与应用系统之间传递的伤害事件。
    struct DamageEvent {
        Entity source{};
        Entity target{};
        int base_damage{};
        int modified_damage{};
        DamageSourceKind source_kind{DamageSourceKind::Attack};
        CombatContext context;
    };

    /// @brief 怪物实体的具体种类。
    struct MonsterIdentity {
        MonsterKind kind;
    };

    /// @brief 实体死亡后写入的击杀归属事件。
    struct KillEvent {
        Entity killer;
        Entity victim;
        MonsterKind monster_kind{};
    };

    /// @brief 用于客户端表现的攻击事件，固化动作开始时的方向和完整时间轴配置。
    struct AttackEvent {
        Entity attacker;
        AttackKind kind{};
        Direction direction{};
        CombatActionID action_id{};
        DeltaTime windup_seconds{};
        DeltaTime active_seconds{};
        DeltaTime recovery_seconds{};
    };

    enum class DeathEntityKind {
        Player,
        Monster,
    };

    /// @brief 用于客户端表现和局内统计的死亡事件。
    struct DeathEvent {
        Entity victim;
        Entity killer;
        DeathEntityKind kind{};
        Position position{};
        Direction direction{};
        std::optional<MonsterKind> monster_kind;
    };

    /// @brief 玩家局内等级、经验和待选祝福次数。
    struct PlayerProgress {
        int level;
        int experience;
        int experience_to_next_level;
        int pending_upgrade_choices;
    };

    /// @brief 一项已持有祝福及其等级。
    struct BlessingStack {
        BlessingID blessing_id = BlessingID::BurnOnHit;
        int level = 1;
    };

    /// @brief 玩家已拥有的全部祝福。
    struct BlessingInventory {
        std::vector<BlessingStack> blessings;
    };

    /// @brief 伤害实际落地后发出的事件，供祝福触发系统消费。
    struct DamageAppliedEvent {
        Entity source{};
        Entity target{};
        int amount{};
        DamageSourceKind source_kind{DamageSourceKind::Attack};
        CombatContext context;
    };

    /// @brief 可叠加的燃烧持续伤害状态。
    struct BurnStatus {
        Entity source{};
        DeltaTime remaining_seconds{0.0f};
        DeltaTime tick_interval_seconds{gameplay_config::blessing::burn_on_hit::TickInterval};
        DeltaTime tick_timer_seconds{0.0f};
        int damage_per_tick{};
    };

    /// @brief 冻结控制状态的剩余时间。
    struct FreezeStatus {
        DeltaTime remaining_seconds{0.0f};
        DeltaTime tick_interval_seconds{gameplay_config::blessing::freeze_on_hit::TickInterval};
        DeltaTime tick_timer_seconds{0.0f};
        int damage_per_tick{};
        Entity source{};
    };

    /// @brief 毒池施加的单实例持续伤害；重复进入只刷新持续时间，不重置 tick 计时。
    struct PoisonStatus {
        Entity source{};
        DeltaTime remaining_seconds{};
        DeltaTime tick_interval_seconds{};
        DeltaTime tick_timer_seconds{};
        int damage_per_tick{};
        bool armor_decreased{false};
    };

    /// @brief 沼泽施加的短时移动倍率；影响普通移动和怪物 AI，不影响冲刺。
    struct SwampStatus {
        DeltaTime remaining_seconds{};
        float movement_multiplier{1.0f};
    };

    struct ArmorBreakStatus {
        DeltaTime remaining_seconds{};
        int armor_break_number{};
        bool armor_decreased{false};
    };

    struct SoulHarvestStatus {
        DeltaTime remaining_seconds{};
        float move_speed_bonus{};
    };

    /// @brief 实体当前承受的持续状态集合。
    struct StatusEffects {
        std::vector<BurnStatus> burns;
        std::optional<FreezeStatus> freeze;
        std::optional<PoisonStatus> poison;
        std::optional<SwampStatus> swamp;
        std::optional<ArmorBreakStatus> armor_break;
        std::optional<SoulHarvestStatus> soul_harvest;
    };

    /// @brief 投射物的飞行距离、伤害和攻击归属。
    struct Projectile {
        int damage{};
        float current_distance{};
        float max_distance{};
        CombatContext context;
    };

    /// @brief 远程怪物保持安全距离的 AI 配置。
    struct KitingAI {
        float retreat_distance;
    };

    enum class BossAbilityKind {
        None,
        TripleDash,
        RadialProjectile,
        Tornado,
    };

    enum class BossPhase {
        One,
        Two,
    };

    struct BossAbilityState {
        BossPhase phase{BossPhase::One};
        BossAbilityKind kind{BossAbilityKind::None};
        AttackPhase action_phase{AttackPhase::Idle};
        DeltaTime remaining_seconds{};
        DeltaTime cooldown_remaining_seconds{};
        std::uint32_t sequence_index{};
        Entity target{NullEntity};
        Position locked_target_position{};
        CombatEffectID ability_id{InvalidEffectID};
        std::vector<Entity> hit_targets{};
        bool invulnerable{false};
        float travelled_distance{};
        BossAbilityKind next_kind{BossAbilityKind::TripleDash};
    };

    struct PathFollowing {
        Entity target{};
        std::vector<Position> waypoints;
        std::size_t current_waypoint{};
        Position target_position{};
    };

    /// @brief 陷阱类型及上一 tick 的范围内目标，用于区分尖刺进入与持续停留。
    struct Trap {
        TrapKind kind{};
        std::vector<Entity> active_targets{};
    };

    enum class ObstacleKind {
        Generic,
        RewardFountain,
        Shop,
    };

    struct Obstacle {
        ObstacleKind kind{};
    };

    enum class CollisionShape {
        Circle,
    };

    enum class CollisionCategory : std::uint32_t {
        Player = 1u << 0,
        Monster = 1u << 1,
        PlayerProjectile = 1u << 2,
        MonsterProjectile = 1u << 3,
        Obstacle = 1u << 4,
        Trap = 1u << 5,
    };

    using CollisionMask = std::uint32_t;

    /// @brief 合并两个碰撞类别为碰撞掩码。
    constexpr CollisionMask operator|(CollisionCategory lhs, CollisionCategory rhs) {
        return static_cast<CollisionMask>(lhs) | static_cast<CollisionMask>(rhs);
    }

    /// @brief 将碰撞类别加入已有碰撞掩码。
    constexpr CollisionMask operator|(CollisionMask lhs, CollisionCategory rhs) {
        return lhs | static_cast<CollisionMask>(rhs);
    }

    /// @brief 将碰撞类别加入已有碰撞掩码。
    constexpr CollisionMask operator|(CollisionCategory lhs, CollisionMask rhs) {
        return static_cast<CollisionMask>(lhs) | rhs;
    }

    /// @brief 返回两个碰撞类别的交集掩码。
    constexpr CollisionMask operator&(CollisionCategory lhs, CollisionCategory rhs) {
        return static_cast<CollisionMask>(lhs) & static_cast<CollisionMask>(rhs);
    }

    /// @brief 返回碰撞掩码与类别的交集。
    constexpr CollisionMask operator&(CollisionMask lhs, CollisionCategory rhs) {
        return lhs & static_cast<CollisionMask>(rhs);
    }

    /// @brief 返回碰撞类别与掩码的交集。
    constexpr CollisionMask operator&(CollisionCategory lhs, CollisionMask rhs) {
        return static_cast<CollisionMask>(lhs) & rhs;
    }

    struct Collider {
        CollisionShape shape;
        float radius;
        CollisionCategory category;
        CollisionMask collision_mask;
    };

    /// @brief 返回两个碰撞体是否分别属于玩家与怪物阵营。
    constexpr bool are_opposing_characters(const Collider& lhs, const Collider& rhs) {
        return (lhs.category == CollisionCategory::Player && rhs.category == CollisionCategory::Monster) ||
            (lhs.category == CollisionCategory::Monster && rhs.category == CollisionCategory::Player);
    }
}
