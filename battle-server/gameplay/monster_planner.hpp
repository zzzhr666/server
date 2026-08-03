#pragma once

#include "ecs/component/components.hpp"
#include "monster_kind.hpp"

namespace battle {
    constexpr float MeleeMonsterAttackRange = 0.7f;
    constexpr ecs::DeltaTime MeleeMonsterAttackCooldown{1.6f};

    constexpr float RangedMonsterAttackRange = 8.0f;
    constexpr ecs::DeltaTime RangedMonsterAttackCooldown{1.4f};

    struct MonsterDefinition {
        MonsterKind kind{};
        int base_health{};
        float base_move_speed{};
        ecs::AttackDefinition base_attack{};
        std::optional<ecs::KitingAI> kiting_ai;
    };

    MonsterDefinition monster_definition(MonsterKind kind);
}
