#include "monster_planner.hpp"

battle::MonsterDefinition battle::monster_definition(MonsterKind kind) {
    switch (kind) {
    case MonsterKind::Melee:
        return {
            .kind = MonsterKind::Melee,
            .base_health = 50,
            .base_move_speed = 3.0f,
        };
    default:
        return {};
    }
}
