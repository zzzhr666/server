#include "weapon.hpp"

battle::WeaponDefinition battle::weapon_definition(WeaponKind kind) {
    switch (kind) {
    case WeaponKind::Sword:
        return {
            .kind = WeaponKind::Sword,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = 25,
                .range = 3.0f,
                .cooldown_seconds = ecs::DeltaTime{0.22f},
                .projectile_speed = 0.0f,
            },
        };
    case WeaponKind::Dagger:
        return {
            .kind = WeaponKind::Dagger,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = 14,
                .range = 2.4f,
                .cooldown_seconds = ecs::DeltaTime{0.12f},
                .projectile_speed = 0.0f,
            },
        };
    case WeaponKind::Axe:
        return {
            .kind = WeaponKind::Axe,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = 40,
                .range = 3.5f,
                .cooldown_seconds = ecs::DeltaTime{0.45f},
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
