#pragma once

#include "ecs/component/components.hpp"
#include "monster_kind.hpp"

namespace battle {
    struct MonsterDefinition {
        MonsterKind kind{};
        int base_health{};
        float base_move_speed{};
        ecs::AttackDefinition base_attack{};
        std::optional<ecs::KitingAI> kiting_ai;
    };

    MonsterDefinition monster_definition(MonsterKind kind);
}
