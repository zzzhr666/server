#include "weapon.hpp"

battle::WeaponDefinition battle::weapon_definition(WeaponKind kind) {
    switch (kind) {
    case WeaponKind::Sword:
        return {
            .kind = WeaponKind::Sword,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = SwordAttackDamage,
                .range = 3.0f,
                .cooldown_seconds = SwordAttackCooldown,
                .projectile_speed = 0.0f,
            },
        };
    case WeaponKind::Dagger:
        return {
            .kind = WeaponKind::Dagger,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = DaggerAttackDamage,
                .range = 2.4f,
                .cooldown_seconds = DaggerAttackCooldown,
                .projectile_speed = 0.0f,
            },
        };
    case WeaponKind::Axe:
        return {
            .kind = WeaponKind::Axe,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = AxeAttackDamage,
                .range = 3.5f,
                .cooldown_seconds = AxeAttackCooldown,
                .projectile_speed = 0.0f,
            },
        };
    case WeaponKind::Bow:
        return {
            .kind = WeaponKind::Bow,
            .attack = {
                .kind = ecs::AttackKind::Projectile,
                .damage = BowAttackDamage,
                .range = 15.0f,
                .cooldown_seconds = BowAttackCooldown,
                .projectile_speed = 25.0f,
                .projectile_hit_radius = BowProjectileHitRadius,
            }
        };
    default:
        return {};
    }
}

std::optional<battle::WeaponKind> battle::weapon_kind_from_string(std::string_view value) {
    if (value == "sword") {
        return WeaponKind::Sword;
    }
    if (value == "dagger") {
        return WeaponKind::Dagger;
    }
    if (value == "axe") {
        return WeaponKind::Axe;
    }
    if (value == "bow") {
        return WeaponKind::Bow;
    }

    return std::nullopt;
}

std::string battle::weapon_kind_to_string(WeaponKind kind) {
    switch (kind) {
    case WeaponKind::Sword:
        return "sword";
    case WeaponKind::Dagger:
        return "dagger";
    case WeaponKind::Axe:
        return "axe";
    case WeaponKind::Bow:
        return "bow";
    }
    return {};
}
