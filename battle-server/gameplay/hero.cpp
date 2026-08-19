#include "hero.hpp"

battle::HeroDefinition battle::hero_definition(HeroKind kind) {
    switch (kind) {
    case HeroKind::Fire:
        return {
            .kind = HeroKind::Fire,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = FireAttackDamage,
                .range = 3.0f,
                .cooldown_seconds = FireAttackCooldown,
                .windup_seconds = FireAttackWindup,
                .active_seconds = FireAttackActive,
                .recovery_seconds = FireAttackRecovery,
                .movement_multiplier = FireAttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
        };
    case HeroKind::Ice:
        return {
            .kind = HeroKind::Ice,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = IceAttackDamage,
                .range = 2.4f,
                .cooldown_seconds = IceAttackCooldown,
                .windup_seconds = IceAttackWindup,
                .active_seconds = IceAttackActive,
                .recovery_seconds = IceAttackRecovery,
                .movement_multiplier = IceAttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
        };
    case HeroKind::Rock:
        return {
            .kind = HeroKind::Rock,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = RockAttackDamage,
                .range = 3.5f,
                .cooldown_seconds = RockAttackCooldown,
                .windup_seconds = RockAttackWindup,
                .active_seconds = RockAttackActive,
                .recovery_seconds = RockAttackRecovery,
                .movement_multiplier = RockAttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
        };
    case HeroKind::Nature:
        return {
            .kind = HeroKind::Nature,
            .attack = {
                .kind = ecs::AttackKind::Projectile,
                .damage = NatureAttackDamage,
                .range = 15.0f,
                .cooldown_seconds = NatureAttackCooldown,
                .windup_seconds = NatureAttackWindup,
                .active_seconds = NatureAttackActive,
                .recovery_seconds = NatureAttackRecovery,
                .movement_multiplier = NatureAttackMovementMultiplier,
                .projectile_speed = 25.0f,
                .projectile_hit_radius = NatureProjectileHitRadius,
            }
        };
    default:
        return {};
    }
}

std::optional<battle::HeroKind> battle::hero_kind_from_string(std::string_view value) {
    if (value == "fire") {
        return HeroKind::Fire;
    }
    if (value == "ice") {
        return HeroKind::Ice;
    }
    if (value == "rock") {
        return HeroKind::Rock;
    }
    if (value == "nature") {
        return HeroKind::Nature;
    }

    return std::nullopt;
}

std::string battle::hero_kind_to_string(HeroKind kind) {
    switch (kind) {
    case HeroKind::Fire:
        return "fire";
    case HeroKind::Ice:
        return "ice";
    case HeroKind::Rock:
        return "rock";
    case HeroKind::Nature:
        return "nature";
    }
    return {};
}
