#pragma once
#include <optional>
#include <string>
#include <string_view>

#include "ecs/component/components.hpp"


namespace battle {
    constexpr int SwordAttackDamage = 23;
    constexpr ecs::DeltaTime SwordAttackCooldown{0.34f};
    constexpr int DaggerAttackDamage = 13;
    constexpr ecs::DeltaTime DaggerAttackCooldown{0.20f};
    constexpr int AxeAttackDamage = 38;
    constexpr ecs::DeltaTime AxeAttackCooldown{0.62f};
    constexpr int BowAttackDamage = 32;
    constexpr ecs::DeltaTime BowAttackCooldown{0.30f};
    constexpr float BowProjectileHitRadius = 0.85f;

    /// @brief 当前支持的初始武器类型。
    enum class WeaponKind {
        Sword,
        Dagger,
        Axe,
        Bow
    };

    /// @brief 武器类型与其 ECS 攻击定义的对应关系。
    struct WeaponDefinition {
        WeaponKind kind{};
        ecs::AttackDefinition attack;
    };

    /// @brief 返回指定武器的权威攻击数值。
    WeaponDefinition weapon_definition(WeaponKind kind);

    /// @brief 将匹配协议中的武器文本解析为武器类型。
    std::optional<WeaponKind> weapon_kind_from_string(std::string_view value);

    /// @brief 将武器类型转换为协议使用的文本。
    std::string weapon_kind_to_string(WeaponKind kind);
}
