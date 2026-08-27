#include "boss_ability_system.hpp"

#include <algorithm>
#include <array>
#include <cstdint>
#include <cmath>
#include <limits>

#include "ecs/world.hpp"
#include "gameplay/gameplay_config.hpp"

namespace {
    battle::ecs::Entity find_nearest_player(battle::ecs::World& world, battle::ecs::Position origin) {
        float nearest_distance = std::numeric_limits<float>::max();
        battle::ecs::Entity nearest_entity{battle::ecs::NullEntity};
        for (const auto player_entity : world.registry().pool<battle::ecs::PlayerController>().entities()) {
            const auto* player_transform = world.registry().try_get<battle::ecs::Transform>(player_entity);
            if (!player_transform) {
                continue;
            }
            const auto distance = std::sqrt(battle::ecs::distance_squared(origin, player_transform->position));
            if (distance < nearest_distance) {
                nearest_distance = distance;
                nearest_entity = player_entity;
            }
        }
        return nearest_entity;
    }

    struct TripleDashConfig {
        battle::ecs::DeltaTime windup;
        float speed;
        float distance;
        battle::ecs::DeltaTime recovery;
        battle::ecs::DeltaTime cooldown;
        int damage;
        float hit_radius;
        std::uint32_t dash_count;
    };

    struct RadialProjectileConfig {
        battle::ecs::DeltaTime windup;
        battle::ecs::DeltaTime interval;
        battle::ecs::DeltaTime recovery;
        battle::ecs::DeltaTime cooldown;
        std::size_t volley_count;
        float speed;
        float range;
        int damage;
        float hit_radius;
    };

    struct TornadoConfig {
        battle::ecs::DeltaTime windup;
        battle::ecs::DeltaTime active;
        battle::ecs::DeltaTime recovery;
        battle::ecs::DeltaTime cooldown;
        float radius;
        int damage;
    };

    TripleDashConfig make_triple_dash_config(battle::ecs::BossPhase phase) {
        if (phase == battle::ecs::BossPhase::Two) {
            return {
                .windup = battle::gameplay_config::monster::boss::triple_dash::phase_two::Windup,
                .speed = battle::gameplay_config::monster::boss::triple_dash::phase_two::Speed,
                .distance = battle::gameplay_config::monster::boss::triple_dash::phase_two::Distance,
                .recovery = battle::gameplay_config::monster::boss::triple_dash::phase_two::Recovery,
                .cooldown = battle::gameplay_config::monster::boss::triple_dash::phase_two::Cooldown,
                .damage = battle::gameplay_config::monster::boss::triple_dash::phase_two::Damage,
                .hit_radius = battle::gameplay_config::monster::boss::triple_dash::phase_two::HitRadius,
                .dash_count = battle::gameplay_config::monster::boss::triple_dash::phase_two::DashCount,
            };
        }
        return {
            .windup = battle::gameplay_config::monster::boss::triple_dash::phase_one::Windup,
            .speed = battle::gameplay_config::monster::boss::triple_dash::phase_one::Speed,
            .distance = battle::gameplay_config::monster::boss::triple_dash::phase_one::Distance,
            .recovery = battle::gameplay_config::monster::boss::triple_dash::phase_one::Recovery,
            .cooldown = battle::gameplay_config::monster::boss::triple_dash::phase_one::Cooldown,
            .damage = battle::gameplay_config::monster::boss::triple_dash::phase_one::Damage,
            .hit_radius = battle::gameplay_config::monster::boss::triple_dash::phase_one::HitRadius,
            .dash_count = battle::gameplay_config::monster::boss::triple_dash::phase_one::DashCount,
        };
    }

    RadialProjectileConfig make_radial_projectile_config(battle::ecs::BossPhase phase) {
        if (phase == battle::ecs::BossPhase::Two) {
            return {
                .windup = battle::gameplay_config::monster::boss::radial_projectile::phase_two::Windup,
                .interval = battle::gameplay_config::monster::boss::radial_projectile::phase_two::Interval,
                .recovery = battle::gameplay_config::monster::boss::radial_projectile::phase_two::Recovery,
                .cooldown = battle::gameplay_config::monster::boss::radial_projectile::phase_two::Cooldown,
                .volley_count = battle::gameplay_config::monster::boss::radial_projectile::phase_two::VolleyCount,
                .speed = battle::gameplay_config::monster::boss::radial_projectile::phase_two::Speed,
                .range = battle::gameplay_config::monster::boss::radial_projectile::phase_two::Range,
                .damage = battle::gameplay_config::monster::boss::radial_projectile::phase_two::Damage,
                .hit_radius = battle::gameplay_config::monster::boss::radial_projectile::phase_two::HitRadius,
            };
        }
        return {
            .windup = battle::gameplay_config::monster::boss::radial_projectile::phase_one::Windup,
            .interval = battle::gameplay_config::monster::boss::radial_projectile::phase_one::Interval,
            .recovery = battle::gameplay_config::monster::boss::radial_projectile::phase_one::Recovery,
            .cooldown = battle::gameplay_config::monster::boss::radial_projectile::phase_one::Cooldown,
            .volley_count = battle::gameplay_config::monster::boss::radial_projectile::phase_one::VolleyCount,
            .speed = battle::gameplay_config::monster::boss::radial_projectile::phase_one::Speed,
            .range = battle::gameplay_config::monster::boss::radial_projectile::phase_one::Range,
            .damage = battle::gameplay_config::monster::boss::radial_projectile::phase_one::Damage,
            .hit_radius = battle::gameplay_config::monster::boss::radial_projectile::phase_one::HitRadius,
        };
    }

    TornadoConfig make_tornado_config(battle::ecs::BossPhase phase) {
        if (phase == battle::ecs::BossPhase::Two) {
            return {
                .windup = battle::gameplay_config::monster::boss::tornado::phase_two::Windup,
                .active = battle::gameplay_config::monster::boss::tornado::phase_two::Active,
                .recovery = battle::gameplay_config::monster::boss::tornado::phase_two::Recovery,
                .cooldown = battle::gameplay_config::monster::boss::tornado::phase_two::Cooldown,
                .radius = battle::gameplay_config::monster::boss::tornado::phase_two::Radius,
                .damage = battle::gameplay_config::monster::boss::tornado::phase_two::Damage,
            };
        }
        return {
            .windup = battle::gameplay_config::monster::boss::tornado::phase_one::Windup,
            .active = battle::gameplay_config::monster::boss::tornado::phase_one::Active,
            .recovery = battle::gameplay_config::monster::boss::tornado::phase_one::Recovery,
            .cooldown = battle::gameplay_config::monster::boss::tornado::phase_one::Cooldown,
            .radius = battle::gameplay_config::monster::boss::tornado::phase_one::Radius,
            .damage = battle::gameplay_config::monster::boss::tornado::phase_one::Damage,
        };
    }

    bool lock_target_position(battle::ecs::World& world, battle::ecs::BossAbilityState& ability) {
        const auto* target_transform = world.registry().try_get<battle::ecs::Transform>(ability.target);
        if (!target_transform) {
            return false;
        }
        ability.locked_target_position = target_transform->position;
        return true;
    }

    void enter_triple_dash_windup(battle::ecs::BossAbilityState& ability, battle::ecs::Transform& transform,
                                  const TripleDashConfig& config) {
        ability.hit_targets.clear();
        ability.action_phase = battle::ecs::AttackPhase::Windup;
        ability.kind = battle::ecs::BossAbilityKind::TripleDash;
        ability.remaining_seconds = config.windup;
        ability.travelled_distance = 0.0f;
        const float delta_x = ability.locked_target_position.x - transform.position.x;
        const float delta_y = ability.locked_target_position.y - transform.position.y;
        const float distance = std::sqrt(delta_x * delta_x + delta_y * delta_y);
        if (distance > 0.01f) {
            transform.direction = {.x = delta_x / distance, .y = delta_y / distance};
        }
    }

    float distance_to_boundary(battle::ecs::Position position, battle::ecs::Direction direction,
                               const battle::ecs::WorldBounds& bounds, float radius) {
        float distance = std::numeric_limits<float>::max();
        if (direction.x > 0.0001f) {
            distance = std::min(distance, (bounds.max_x - radius - position.x) / direction.x);
        } else if (direction.x < -0.0001f) {
            distance = std::min(distance, (bounds.min_x + radius - position.x) / direction.x);
        }
        if (direction.y > 0.0001f) {
            distance = std::min(distance, (bounds.max_y - radius - position.y) / direction.y);
        } else if (direction.y < -0.0001f) {
            distance = std::min(distance, (bounds.min_y + radius - position.y) / direction.y);
        }
        return std::max(distance, 0.0f);
    }

    void enter_radial_projectile_windup(battle::ecs::BossAbilityState& ability,
                                        const RadialProjectileConfig& config) {
        ability.hit_targets.clear();
        ability.action_phase = battle::ecs::AttackPhase::Windup;
        ability.kind = battle::ecs::BossAbilityKind::RadialProjectile;
        ability.remaining_seconds = config.windup;
        ability.sequence_index = 0;
    }

    void enter_tornado_windup(battle::ecs::BossAbilityState& ability, const TornadoConfig& config) {
        ability.hit_targets.clear();
        ability.kind = battle::ecs::BossAbilityKind::Tornado;
        ability.action_phase = battle::ecs::AttackPhase::Windup;
        ability.remaining_seconds = config.windup;
    }

    void finish_boss_ability(battle::ecs::BossAbilityState& ability, battle::ecs::Velocity& velocity,
                             battle::ecs::DeltaTime cooldown) {
        velocity.x = 0.0f;
        velocity.y = 0.0f;
        ability.action_phase = battle::ecs::AttackPhase::Idle;
        ability.kind = battle::ecs::BossAbilityKind::None;
        ability.remaining_seconds = battle::ecs::DeltaTime{0};
        ability.cooldown_remaining_seconds = cooldown;
        ability.sequence_index = 0;
        ability.target = battle::ecs::NullEntity;
        ability.locked_target_position = {};
        ability.ability_id = battle::ecs::InvalidEffectID;
        ability.travelled_distance = 0.0f;
        ability.hit_targets.clear();
    }

    float point_segment_distance_squared(battle::ecs::Position point, battle::ecs::Position segment_start,
                                         battle::ecs::Position segment_end) {
        const float segment_x = segment_end.x - segment_start.x;
        const float segment_y = segment_end.y - segment_start.y;
        const float point_x = point.x - segment_start.x;
        const float point_y = point.y - segment_start.y;
        const float segment_length_squared = segment_x * segment_x + segment_y * segment_y;
        if (segment_length_squared <= 0.000001f) {
            return battle::ecs::distance_squared(point, segment_start);
        }
        const float projection = std::clamp(
            (point_x * segment_x + point_y * segment_y) / segment_length_squared, 0.0f, 1.0f);
        const battle::ecs::Position closest{
            .x = segment_start.x + segment_x * projection,
            .y = segment_start.y + segment_y * projection,
        };
        return battle::ecs::distance_squared(point, closest);
    }

    void resolve_triple_dash_hits(battle::ecs::World& world, battle::ecs::Entity boss_entity,
                                  battle::ecs::Position start,
                                  battle::ecs::Position end, battle::ecs::BossAbilityState& ability,
                                  const TripleDashConfig& config) {
        const auto* boss_collider = world.registry().try_get<battle::ecs::Collider>(boss_entity);
        if (!boss_collider) {
            return;
        }
        const float broad_phase_radius = config.hit_radius;
        const battle::ecs::Position min_corner{
            .x = std::min(start.x, end.x) - broad_phase_radius,
            .y = std::min(start.y, end.y) - broad_phase_radius,
        };
        const battle::ecs::Position max_corner{
            .x = std::max(start.x, end.x) + broad_phase_radius,
            .y = std::max(start.y, end.y) + broad_phase_radius,
        };
        for (const battle::ecs::Entity entity : world.spatial_index().query_aabb(min_corner, max_corner)) {
            if (entity == boss_entity || !world.registry().has<battle::ecs::PlayerController>(entity)) {
                continue;
            }
            if (std::ranges::find(ability.hit_targets, entity) != ability.hit_targets.end()) {
                continue;
            }
            const auto* transform = world.registry().try_get<battle::ecs::Transform>(entity);
            const auto* collider = world.registry().try_get<battle::ecs::Collider>(entity);
            const auto* health = world.registry().try_get<battle::ecs::Health>(entity);
            if (!transform || !collider || !health || health->current_health <= 0 ||
                !battle::ecs::are_opposing_characters(*boss_collider, *collider)) {
                continue;
            }
            const float collision_radius = config.hit_radius + collider->radius;
            if (point_segment_distance_squared(transform->position, start, end) > collision_radius * collision_radius) {
                continue;
            }
            world.add_damage_event(battle::ecs::DamageEvent{
                .source = boss_entity,
                .target = entity,
                .base_damage = config.damage,
                .modified_damage = config.damage,
                .source_kind = battle::ecs::DamageSourceKind::Attack,
                .context = battle::ecs::CombatContext{
                    .owner = boss_entity,
                    .emitter = boss_entity,
                    .effect_id = ability.ability_id,
                }
            });
            ability.hit_targets.emplace_back(entity);
        }
    }

    void resolve_tornado_hits(battle::ecs::World& world, battle::ecs::Entity boss_entity,
                              battle::ecs::BossAbilityState& ability, const TornadoConfig& config) {
        const auto* transform = world.registry().try_get<battle::ecs::Transform>(boss_entity);
        if (!transform) {
            return;
        }
        for (battle::ecs::Entity entity : world.spatial_index().query_circle(
                 transform->position, config.radius)) {
            if (!world.is_living_player(entity)) {
                continue;
            }
            if (std::ranges::find(ability.hit_targets, entity) != ability.hit_targets.end()) {
                continue;
            }
            const auto* player_transform = world.registry().try_get<battle::ecs::Transform>(entity);
            const auto* player_collider = world.registry().try_get<battle::ecs::Collider>(entity);
            if (!player_transform || !player_collider) {
                continue;
            }
            const float radius_sum =
                player_collider->radius + config.radius;
            if (battle::ecs::distance_squared(transform->position, player_transform->position) >=
                radius_sum * radius_sum) {
                continue;
            }
            world.add_damage_event(battle::ecs::DamageEvent{
                .source = boss_entity,
                .target = entity,
                .base_damage = config.damage,
                .modified_damage = config.damage,
                .source_kind = battle::ecs::DamageSourceKind::Attack,
                .context = battle::ecs::CombatContext{
                    .owner = boss_entity,
                    .emitter = boss_entity,
                    .effect_id = ability.ability_id,
                }
            });
            ability.hit_targets.emplace_back(entity);
        }
    }

    constexpr double sqrt_constexpr(double x) {
        if (x <= 0.0) {
            return 0.0;
        }

        double guess = x;

        for (int i = 0; i < 20; ++i) {
            guess = 0.5 * (guess + x / guess);
        }

        return guess;
    }

    void spawn_radial_projectiles(battle::ecs::World& world, battle::ecs::Entity boss_entity,
                                  const battle::ecs::BossAbilityState& ability,
                                  const RadialProjectileConfig& config) {
        constexpr float sqrt2 = sqrt_constexpr(2);
        constexpr std::array<battle::ecs::Direction, 8> directions{
            {
                {.x = 1, .y = 0}, {.x = -1, .y = 0}, {.x = 0, .y = 1}, {.x = 0, .y = -1},
                {.x = 1 / sqrt2, .y = 1 / sqrt2}, {.x = -1 / sqrt2, .y = 1 / sqrt2},
                {.x = 1 / sqrt2, .y = -1 / sqrt2}, {.x = -1 / sqrt2, .y = -1 / sqrt2},
            }
        };
        const auto* transform = world.registry().try_get<battle::ecs::Transform>(boss_entity);
        if (!transform) {
            return;
        }
        for (const auto& direction : directions) {
            world.create_projectile(battle::ecs::CreateProjectileConfig{
                .position = transform->position,
                .direction = direction,
                .speed = config.speed,
                .damage = config.damage,
                .max_distance = config.range,
                .hit_radius = config.hit_radius,
                .context = battle::ecs::CombatContext{
                    .owner = boss_entity,
                    .emitter = boss_entity,
                    .effect_id = ability.ability_id,
                },
            });
        }
    }

    battle::ecs::BossAbilityKind next_kind(battle::ecs::BossAbilityKind kind) {
        switch (kind) {
        case battle::ecs::BossAbilityKind::TripleDash:
            return battle::ecs::BossAbilityKind::RadialProjectile;
        case battle::ecs::BossAbilityKind::RadialProjectile:
            return battle::ecs::BossAbilityKind::Tornado;
        case battle::ecs::BossAbilityKind::Tornado: //NOLINT
            return battle::ecs::BossAbilityKind::TripleDash;
        default:
            return battle::ecs::BossAbilityKind::TripleDash;
        }
    }
}

void battle::ecs::boss_ability_system(World& world, DeltaTime delta_time) {
    for (const auto entity : world.registry().pool<BossAbilityState>().entities()) {
        const auto* monster_identity = world.registry().try_get<MonsterIdentity>(entity);
        if (!monster_identity || monster_identity->kind != MonsterKind::Boss) {
            continue;
        }
        auto* transform = world.registry().try_get<Transform>(entity);
        auto* velocity = world.registry().try_get<Velocity>(entity);
        auto* ability = world.registry().try_get<BossAbilityState>(entity);
        auto* health = world.registry().try_get<Health>(entity);
        if (!transform || !velocity || !ability || !health) {
            continue;
        }
        ability->remaining_seconds -= delta_time;
        if (ability->remaining_seconds <= DeltaTime{0}) {
            ability->remaining_seconds = DeltaTime{0};
        }
        ability->cooldown_remaining_seconds -= delta_time;
        if (ability->cooldown_remaining_seconds <= DeltaTime{0}) {
            ability->cooldown_remaining_seconds = DeltaTime{0};
        }
        if (health && ability->phase == BossPhase::One && health->current_health * 2 <= health->max_health) {
            ability->phase = BossPhase::Two;
        }
        const auto dash_config = make_triple_dash_config(ability->phase);
        const auto radial_projectile_cfg = make_radial_projectile_config(ability->phase);
        const auto tornado_cfg = make_tornado_config(ability->phase);
        if (ability->kind == BossAbilityKind::None) {
            if (ability->cooldown_remaining_seconds > DeltaTime{0}) {
                velocity->x = 0.0f;
                velocity->y = 0.0f;
                continue;
            }
            if (ability->target == NullEntity) {
                ability->target = find_nearest_player(world, transform->position);
            }
            if (!lock_target_position(world, *ability)) {
                finish_boss_ability(*ability, *velocity, battle::ecs::DeltaTime{0.0f});
                continue;
            }
            ability->sequence_index = 0;
            ability->ability_id = world.create_combat_effect();
            switch (ability->next_kind) {
            case BossAbilityKind::TripleDash: {
                enter_triple_dash_windup(*ability, *transform, dash_config);
                break;
            }
            case BossAbilityKind::RadialProjectile: {
                enter_radial_projectile_windup(*ability, radial_projectile_cfg);
                break;
            }
            case BossAbilityKind::Tornado: {
                enter_tornado_windup(*ability, tornado_cfg);
                break;
            }
            case BossAbilityKind::None: {
                enter_triple_dash_windup(*ability, *transform, dash_config);
                break;
            }
            }
            ability->next_kind = next_kind(ability->kind);
            continue;
        }
        if (ability->action_phase == AttackPhase::Windup && ability->remaining_seconds <= DeltaTime{0}) {
            ability->action_phase = AttackPhase::Active;
            if (ability->kind == BossAbilityKind::Tornado) {
                ability->remaining_seconds = tornado_cfg.active;
            } else {
                ability->remaining_seconds = DeltaTime{0};
            }
        }
        if (ability->kind == BossAbilityKind::TripleDash && ability->action_phase == AttackPhase::Active) {
            const float delta_seconds = delta_time.count();
            const float remaining_distance = dash_config.distance - ability->travelled_distance;
            const float max_step_distance = dash_config.speed * delta_seconds;
            const auto* collider = world.registry().try_get<Collider>(entity);
            const float direction_length = std::sqrt(transform->direction.x * transform->direction.x +
                transform->direction.y * transform->direction.y);
            if (!collider || direction_length <= 0.01f || delta_seconds <= 0.0f || remaining_distance <= 0.0f) {
                velocity->x = 0.0f;
                velocity->y = 0.0f;
                ability->action_phase = AttackPhase::Recovery;
                ability->remaining_seconds = dash_config.recovery;
                continue;
            }
            const Direction dash_direction{
                .x = transform->direction.x / direction_length,
                .y = transform->direction.y / direction_length,
            };
            const float boundary_distance = distance_to_boundary(transform->position, dash_direction,
                                                                 world.world_bounds(), collider->radius);
            const float step_distance = std::min({max_step_distance, remaining_distance, boundary_distance});
            if (step_distance <= 0.0f) {
                velocity->x = 0.0f;
                velocity->y = 0.0f;
                ability->action_phase = AttackPhase::Recovery;
                ability->remaining_seconds = dash_config.recovery;
                continue;
            }
            const float step_speed = step_distance / delta_seconds;
            velocity->x = dash_direction.x * step_speed;
            velocity->y = dash_direction.y * step_speed;
            ability->travelled_distance += step_distance;
            Position start = transform->position;
            Position end = {
                .x = start.x + velocity->x * delta_time.count(),
                .y = start.y + velocity->y * delta_time.count(),
            };
            resolve_triple_dash_hits(world, entity, start, end, *ability, dash_config);
            continue;
        }
        if (ability->kind == BossAbilityKind::TripleDash &&
            ability->action_phase == AttackPhase::Recovery &&
            ability->remaining_seconds <= DeltaTime{0}) {
            if (ability->sequence_index + 1u < dash_config.dash_count) {
                if (!lock_target_position(world, *ability)) {
                    finish_boss_ability(*ability, *velocity, dash_config.cooldown);
                    continue;
                }
                ability->sequence_index += 1;
                enter_triple_dash_windup(*ability, *transform, dash_config);
            } else {
                finish_boss_ability(*ability, *velocity, dash_config.cooldown);
            }
            continue;
        }
        if (ability->kind == BossAbilityKind::RadialProjectile &&
            ability->action_phase == AttackPhase::Active &&
            ability->remaining_seconds <= DeltaTime{0}) {
            spawn_radial_projectiles(world, entity, *ability, radial_projectile_cfg);
            ++ability->sequence_index;
            if (ability->sequence_index < radial_projectile_cfg.volley_count) {
                ability->remaining_seconds = radial_projectile_cfg.interval;
            } else {
                ability->action_phase = AttackPhase::Recovery;
                ability->remaining_seconds = radial_projectile_cfg.recovery;
            }
            continue;
        }
        if (ability->kind == BossAbilityKind::RadialProjectile &&
            ability->action_phase == AttackPhase::Recovery &&
            ability->remaining_seconds <= DeltaTime{0}) {
            finish_boss_ability(*ability, *velocity, radial_projectile_cfg.cooldown);
            continue;
        }
        if (ability->kind == BossAbilityKind::Tornado &&
            ability->action_phase == AttackPhase::Active) {
            if (ability->remaining_seconds <= DeltaTime{0}) {
                ability->action_phase = AttackPhase::Recovery;
                ability->remaining_seconds = tornado_cfg.recovery;
            } else {
                resolve_tornado_hits(world, entity, *ability, tornado_cfg);
            }
            continue;
        }
        if (ability->kind == BossAbilityKind::Tornado &&
            ability->action_phase == AttackPhase::Recovery &&
            ability->remaining_seconds <= DeltaTime{0}) {
            finish_boss_ability(*ability, *velocity, tornado_cfg.cooldown);
            continue;
        }
    }
}
