#include "monster_planner.hpp"

battle::MonsterDefinition battle::monster_definition(MonsterKind kind) {
    switch (kind) {
    case MonsterKind::Melee:
        return {
            .kind = MonsterKind::Melee,
            .base_health = 50,
            .base_move_speed = 3.0f,
            .base_attack = ecs::AttackDefinition{
                .kind = ecs::AttackKind::Melee,
                .damage = 10,
                .range = MeleeMonsterAttackRange,
                .cooldown_seconds = MeleeMonsterAttackCooldown,
                .windup_seconds = MeleeMonsterAttackWindup,
                .active_seconds = MeleeMonsterAttackActive,
                .recovery_seconds = MeleeMonsterAttackRecovery,
                .movement_multiplier = MeleeMonsterAttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
        };

        case MonsterKind::Ranged:
        return {
            .kind = MonsterKind::Ranged,
            .base_health = 35,
            .base_move_speed = 3.5f,
            .base_attack = {
                .kind = ecs::AttackKind::Projectile,
                .damage = 12,
                .range = RangedMonsterAttackRange,
                .cooldown_seconds = RangedMonsterAttackCooldown,
                .windup_seconds = RangedMonsterAttackWindup,
                .active_seconds = RangedMonsterAttackActive,
                .recovery_seconds = RangedMonsterAttackRecovery,
                .movement_multiplier = RangedMonsterAttackMovementMultiplier,
                .projectile_speed = RangedMonsterProjectileSpeed,
            },
            .kiting_ai = ecs::KitingAI{
                .retreat_distance = RangedMonsterRetreatDistance,
            }
        };
    default:
        return {};
    }
}
