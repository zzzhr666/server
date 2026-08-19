#pragma once
#include <optional>
#include <string>
#include <string_view>

#include "ecs/component/components.hpp"


namespace battle {
    constexpr int FireAttackDamage = 23;
    constexpr ecs::DeltaTime FireAttackCooldown{0.34f};
    constexpr ecs::DeltaTime FireAttackWindup{0.12f};
    constexpr ecs::DeltaTime FireAttackActive{0.05f};
    constexpr ecs::DeltaTime FireAttackRecovery{0.17f};
    constexpr float FireAttackMovementMultiplier{0.25f};

    constexpr int IceAttackDamage = 13;
    constexpr ecs::DeltaTime IceAttackCooldown{0.20f};
    constexpr ecs::DeltaTime IceAttackWindup{0.06f};
    constexpr ecs::DeltaTime IceAttackActive{0.03f};
    constexpr ecs::DeltaTime IceAttackRecovery{0.11f};
    constexpr float IceAttackMovementMultiplier{0.55f};

    constexpr int RockAttackDamage = 38;
    constexpr ecs::DeltaTime RockAttackCooldown{0.62f};
    constexpr ecs::DeltaTime RockAttackWindup{0.24f};
    constexpr ecs::DeltaTime RockAttackActive{0.08f};
    constexpr ecs::DeltaTime RockAttackRecovery{0.30f};
    constexpr float RockAttackMovementMultiplier{0.0f};

    constexpr int NatureAttackDamage = 32;
    constexpr ecs::DeltaTime NatureAttackCooldown{0.30f};
    constexpr ecs::DeltaTime NatureAttackWindup{0.12f};
    constexpr ecs::DeltaTime NatureAttackActive{0.02f};
    constexpr ecs::DeltaTime NatureAttackRecovery{0.16f};
    constexpr float NatureAttackMovementMultiplier{0.35f};
    constexpr float NatureProjectileHitRadius = 0.85f;

    /// @brief 当前支持的初始英雄类型。
    enum class HeroKind {
        Fire,
        Ice,
        Rock,
        Nature
    };

    /// @brief 英雄类型与其 ECS 攻击定义的对应关系。
    struct HeroDefinition {
        HeroKind kind{};
        ecs::AttackDefinition attack;
    };

    /// @brief 返回指定英雄的权威攻击数值。
    HeroDefinition hero_definition(HeroKind kind);

    /// @brief 将匹配协议中的英雄文本解析为英雄类型。
    std::optional<HeroKind> hero_kind_from_string(std::string_view value);

    /// @brief 将英雄类型转换为协议使用的文本。
    std::string hero_kind_to_string(HeroKind kind);
}
