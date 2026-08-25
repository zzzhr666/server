#include "monster_planner.hpp"
#include "gameplay_config.hpp"

battle::MonsterDefinition battle::monster_definition(MonsterKind kind) {
    switch (kind) {
    case MonsterKind::Melee:
        return {
            .kind = MonsterKind::Melee,
            .base_health = gameplay_config::monster::melee::Health,
            .base_move_speed = gameplay_config::monster::melee::MoveSpeed,
            .base_attack = ecs::AttackDefinition{
                .kind = ecs::AttackKind::Melee,
                .damage = gameplay_config::monster::melee::AttackDamage,
                .range = gameplay_config::monster::melee::AttackRange,
                .cooldown_seconds = gameplay_config::monster::melee::AttackCooldown,
                .windup_seconds = gameplay_config::monster::melee::AttackWindup,
                .active_seconds = gameplay_config::monster::melee::AttackActive,
                .recovery_seconds = gameplay_config::monster::melee::AttackRecovery,
                .movement_multiplier = gameplay_config::monster::melee::AttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
            .soul_reward = gameplay_config::monster::melee::SoulReward,
            .collision_radius = gameplay_config::combat::MonsterCollisionRadius,
        };

        case MonsterKind::Ranged:
        return {
            .kind = MonsterKind::Ranged,
            .base_health = gameplay_config::monster::ranged::Health,
            .base_move_speed = gameplay_config::monster::ranged::MoveSpeed,
            .base_attack = {
                .kind = ecs::AttackKind::Projectile,
                .damage = gameplay_config::monster::ranged::AttackDamage,
                .range = gameplay_config::monster::ranged::AttackRange,
                .cooldown_seconds = gameplay_config::monster::ranged::AttackCooldown,
                .windup_seconds = gameplay_config::monster::ranged::AttackWindup,
                .active_seconds = gameplay_config::monster::ranged::AttackActive,
                .recovery_seconds = gameplay_config::monster::ranged::AttackRecovery,
                .movement_multiplier = gameplay_config::monster::ranged::AttackMovementMultiplier,
                .projectile_speed = gameplay_config::monster::ranged::ProjectileSpeed,
            },
            .kiting_ai = ecs::KitingAI{
                .retreat_distance = gameplay_config::monster::ranged::RetreatDistance,
            },
            .soul_reward = gameplay_config::monster::ranged::SoulReward,
            .collision_radius = gameplay_config::combat::MonsterCollisionRadius,
        };
    case MonsterKind::Boss:
        return {
            .kind = MonsterKind::Boss,
            .base_health = gameplay_config::monster::boss::Health,
            .base_move_speed = gameplay_config::monster::boss::MoveSpeed,
            .base_attack = {},
            .soul_reward = gameplay_config::monster::boss::SoulReward,
            .collision_radius = gameplay_config::monster::boss::CollisionRadius,
        };
    default:
        return {};
    }
}
