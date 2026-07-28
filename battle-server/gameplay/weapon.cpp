#include "weapon.hpp"

battle::WeaponDefinition battle::weapon_definition(WeaponKind kind) {
    switch (kind) {
    case WeaponKind::Sword:
        return {
            .kind = WeaponKind::Sword,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = 30,
                .range = 1.6f,
                .cooldown_seconds = ecs::DeltaTime{0.4f},
                .projectile_speed = 0.0f,
            },
        };
    case WeaponKind::Dagger:
        return {
            .kind = WeaponKind::Dagger,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = 16,
                .range = 1.1f,
                .cooldown_seconds = ecs::DeltaTime{0.18f},
                .projectile_speed = 0.0f,
            },
        };
    case WeaponKind::Axe:
        return {
            .kind = WeaponKind::Axe,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = 55,
                .range = 1.9f,
                .cooldown_seconds = ecs::DeltaTime{0.9f},
                .projectile_speed = 0.0f,
            },
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
    }
    return {};
}
