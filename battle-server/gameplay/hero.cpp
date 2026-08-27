#include "hero.hpp"
#include "gameplay_config.hpp"

battle::HeroDefinition battle::hero_definition(HeroKind kind) {
    switch (kind) {
    case HeroKind::Fire:
        return {
            .kind = HeroKind::Fire,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = gameplay_config::hero::fire::AttackDamage,
                .range = gameplay_config::hero::fire::AttackRange,
                .cooldown_seconds = gameplay_config::hero::fire::AttackCooldown,
                .windup_seconds = gameplay_config::hero::fire::AttackWindup,
                .active_seconds = gameplay_config::hero::fire::AttackActive,
                .recovery_seconds = gameplay_config::hero::fire::AttackRecovery,
                .movement_multiplier = gameplay_config::hero::fire::AttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
        };
    case HeroKind::Ice:
        return {
            .kind = HeroKind::Ice,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = gameplay_config::hero::ice::AttackDamage,
                .range = gameplay_config::hero::ice::AttackRange,
                .cooldown_seconds = gameplay_config::hero::ice::AttackCooldown,
                .windup_seconds = gameplay_config::hero::ice::AttackWindup,
                .active_seconds = gameplay_config::hero::ice::AttackActive,
                .recovery_seconds = gameplay_config::hero::ice::AttackRecovery,
                .movement_multiplier = gameplay_config::hero::ice::AttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
        };
    case HeroKind::Rock:
        return {
            .kind = HeroKind::Rock,
            .attack = {
                .kind = ecs::AttackKind::Melee,
                .damage = gameplay_config::hero::rock::AttackDamage,
                .range = gameplay_config::hero::rock::AttackRange,
                .cooldown_seconds = gameplay_config::hero::rock::AttackCooldown,
                .windup_seconds = gameplay_config::hero::rock::AttackWindup,
                .active_seconds = gameplay_config::hero::rock::AttackActive,
                .recovery_seconds = gameplay_config::hero::rock::AttackRecovery,
                .movement_multiplier = gameplay_config::hero::rock::AttackMovementMultiplier,
                .projectile_speed = 0.0f,
            },
        };
    case HeroKind::Nature:
        return {
            .kind = HeroKind::Nature,
            .attack = {
                .kind = ecs::AttackKind::Projectile,
                .damage = gameplay_config::hero::nature::AttackDamage,
                .range = gameplay_config::hero::nature::AttackRange,
                .cooldown_seconds = gameplay_config::hero::nature::AttackCooldown,
                .windup_seconds = gameplay_config::hero::nature::AttackWindup,
                .active_seconds = gameplay_config::hero::nature::AttackActive,
                .recovery_seconds = gameplay_config::hero::nature::AttackRecovery,
                .movement_multiplier = gameplay_config::hero::nature::AttackMovementMultiplier,
                .projectile_speed = gameplay_config::hero::nature::ProjectileSpeed,
                .projectile_hit_radius = gameplay_config::hero::nature::ProjectileHitRadius,
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
