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
                .range = 1.0f,
                .cooldown_seconds = ecs::DeltaTime{1.0f},
                .projectile_speed = 0.0f,
            },
        };
    default:
        return {};
    }
}
