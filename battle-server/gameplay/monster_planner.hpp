#pragma once

#include "ecs/component/components.hpp"
#include "monster_kind.hpp"

namespace battle {
    constexpr float MeleeMonsterAttackRange = 0.7f;
    constexpr ecs::DeltaTime MeleeMonsterAttackCooldown{1.6f};
    constexpr ecs::DeltaTime MeleeMonsterAttackWindup{0.45f};
    constexpr ecs::DeltaTime MeleeMonsterAttackActive{0.1f};
    constexpr ecs::DeltaTime MeleeMonsterAttackRecovery{1.05f};
    constexpr float MeleeMonsterAttackMovementMultiplier{0.0f};

    constexpr float RangedMonsterAttackRange = 10.5f;
    constexpr ecs::DeltaTime RangedMonsterAttackCooldown{2.0f};
    constexpr ecs::DeltaTime RangedMonsterAttackWindup{0.6f};
    constexpr ecs::DeltaTime RangedMonsterAttackActive{0.05f};
    constexpr ecs::DeltaTime RangedMonsterAttackRecovery{1.35f};
    constexpr float RangedMonsterAttackMovementMultiplier{0.0f};
    constexpr float RangedMonsterProjectileSpeed = 11.0f;
    constexpr float RangedMonsterRetreatDistance = 7.0f;

    struct MonsterDefinition {
        MonsterKind kind{};
        int base_health{};
        float base_move_speed{};
        ecs::AttackDefinition base_attack{};
        std::optional<ecs::KitingAI> kiting_ai;
    };

    MonsterDefinition monster_definition(MonsterKind kind);
}
