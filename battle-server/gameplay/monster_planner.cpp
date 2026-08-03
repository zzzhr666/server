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
                .projectile_speed = 18.0f,
            },
            .kiting_ai = ecs::KitingAI{
                .retreat_distance = 5.0f,
            }
        };
    default:
        return {};
    }
}
