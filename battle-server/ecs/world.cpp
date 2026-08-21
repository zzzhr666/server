#include "world.hpp"

#include <algorithm>
#include <cmath>
#include <utility>

#include "system/attack_resolve_system.hpp"
#include "system/blessing_trigger_system.hpp"
#include "system/damage_modify_system.hpp"
#include "system/damage_system.hpp"
#include "system/dash_resolve_system.hpp"
#include "system/dash_system.hpp"
#include "system/death_system.hpp"
#include "system/hit_resolve_system.hpp"
#include "system/monster_ai_system.hpp"
#include "system/move_resolve_system.hpp"
#include "system/move_system.hpp"
#include "system/pre_damage_blessing_system.hpp"
#include "system/projectile_hit_system.hpp"
#include "system/projectile_range_system.hpp"
#include "system/projectile_spawn_system.hpp"
#include "system/status_effect_system.hpp"
#include "system/trap_system.hpp"

namespace {
    bool is_character(const battle::ecs::Collider& collider) {
        return collider.category == battle::ecs::CollisionCategory::Monster ||
            collider.category == battle::ecs::CollisionCategory::Player;
    }

    bool is_interactable(const battle::ecs::Collider& lhs, const battle::ecs::Collider& rhs) {
        return (lhs.category & rhs.collision_mask) != 0 &&
            (lhs.collision_mask & rhs.category) != 0;
    }

    bool is_overlap(const battle::ecs::Position& center_a, const battle::ecs::Position& center_b, float radius_a,
                    float radius_b) {
        const float dx = center_b.x - center_a.x;
        const float dy = center_b.y - center_a.y;
        const float radius_sum = radius_a + radius_b;
        return dx * dx + dy * dy < radius_sum * radius_sum;
    }

    void normalize_in_boundary(battle::ecs::Position& position, const battle::ecs::WorldBounds& bounds,
                               const battle::ecs::Collider& collider) {
        position.x = std::clamp(position.x, bounds.min_x + collider.radius, bounds.max_x - collider.radius);
        position.y = std::clamp(position.y, bounds.min_y + collider.radius, bounds.max_y - collider.radius);
    }
}

battle::ecs::World::World(WorldBounds bounds, std::uint32_t random_seed, float cell_size)
    : bounds_(bounds), spatial_index_(cell_size), random_engine_(random_seed), percent_distribution_(1, 100),
      next_combat_action_id_(1),
      next_combat_effect_id_(1) {
    damage_events_.reserve(InitialDamageEventCount);
    // 状态先衰减，移动与冲刺随后结算，陷阱在新位置触发，伤害最终进入统一死亡链路。
    system_scheduler_.add_system(move_resolve_system);
    system_scheduler_.add_system(status_effect_system);
    system_scheduler_.add_system(monster_ai_system);
    system_scheduler_.add_system(dash_resolve_system);
    system_scheduler_.add_system(dash_system);
    system_scheduler_.add_system(move_system);
    system_scheduler_.add_system(trap_system);
    system_scheduler_.add_system(projectile_hit_system);
    system_scheduler_.add_system(projectile_range_system);
    system_scheduler_.add_system(attack_resolve_system);
    system_scheduler_.add_system(projectile_spawn_system);
    system_scheduler_.add_system(hit_resolve_system);
    system_scheduler_.add_system(damage_modify_system);
    system_scheduler_.add_system(pre_damage_blessing_system);
    system_scheduler_.add_system(damage_system);
    system_scheduler_.add_system(blessing_trigger_system);
    system_scheduler_.add_system(death_system);
}

battle::ecs::World::World(std::initializer_list<sysFunc> functions, WorldBounds bounds,
                          std::uint32_t random_seed, float cell_size)
    : system_scheduler_(functions), bounds_(bounds), spatial_index_(cell_size), random_engine_(random_seed),
      percent_distribution_(1, 100),
      next_combat_action_id_(1), next_combat_effect_id_(1) {}

battle::ecs::Entity battle::ecs::World::create_player(CreatePlayerConfig config) {
    auto position = resolve_character_spawn_(config.position, {
                                                 .shape = CollisionShape::Circle, .radius = config.collision_radius,
                                                 .category = CollisionCategory::Player,
                                                 .collision_mask = PlayerCollisionMask
                                             });
    if (!position.has_value()) {
        return NullEntity;
    }
    Entity entity = registry_.create();

    registry_.emplace<Transform>(entity, position.value(), Direction{.x = 0.0f, .y = 1.0f});
    registry_.emplace<Velocity>(entity, 0.0f, 0.0f);
    registry_.emplace<MoveRequest>(entity, 0.0f, 0.0f);

    registry_.emplace<AttackRequest>(entity, false);
    registry_.emplace<DashRequest>(entity, false);
    registry_.emplace<MoveIntent>(entity, 0.0f, 0.0f);
    registry_.emplace<Health>(entity, config.max_health, config.max_health);
    registry_.emplace<CharacterStats>(entity, config.move_speed);
    registry_.emplace<PlayerController>(entity);
    registry_.emplace<Dash>(entity, gameplay_config::player::DashCooldown,
                            gameplay_config::player::DashSpeedMultiplier);
    registry_.emplace<DashIntent>(entity, false, 0.0f);
    registry_.emplace<AttackDefinition>(entity, config.attack);
    registry_.emplace<AttackCooldown>(entity, DeltaTime{0.0f});
    registry_.emplace<DashCooldown>(entity, DeltaTime{0.0f});
    registry_.emplace<PlayerProgress>(entity, 1, 0, 100, 0);
    registry_.emplace<BlessingInventory>(entity);
    registry_.emplace<StatusEffects>(entity);
    registry_.emplace<AttackState>(entity);
    registry_.emplace<Collider>(entity, CollisionShape::Circle, config.collision_radius,
                                CollisionCategory::Player, PlayerCollisionMask);
    const auto* transform = registry_.try_get<Transform>(entity);
    if (const auto* collider = registry_.try_get<Collider>(entity); transform && collider) {
        spatial_index_.insert(entity, transform->position, collider->radius);
    }
    return entity;
}

battle::ecs::Entity battle::ecs::World::create_monster(CreateMonsterConfig config) {
    auto position = resolve_character_spawn_(config.position, {
                                                 .shape = CollisionShape::Circle, .radius = config.collision_radius,
                                                 .category = CollisionCategory::Monster,
                                                 .collision_mask = MonsterCollisionMask
                                             });
    if (!position.has_value()) {
        return NullEntity;
    }
    Entity entity = registry_.create();
    registry_.emplace<Transform>(entity, position.value(), Direction{.x = 0.0f, .y = 1.0f});
    registry_.emplace<Velocity>(entity, 0.0f, 0.0f);
    registry_.emplace<Health>(entity, config.max_health, config.max_health);
    registry_.emplace<CharacterStats>(entity, config.move_speed);
    registry_.emplace<MonsterController>(entity);
    registry_.emplace<AttackRequest>(entity, false);
    registry_.emplace<AttackDefinition>(entity, config.attack);
    registry_.emplace<AttackCooldown>(entity, DeltaTime{0.0f});
    registry_.emplace<MonsterIdentity>(entity, config.kind);
    registry_.emplace<StatusEffects>(entity);
    registry_.emplace<AttackState>(entity);
    registry_.emplace<Collider>(entity, CollisionShape::Circle, config.collision_radius,
                                CollisionCategory::Monster, MonsterCollisionMask);
    if (config.kiting_ai.has_value()) {
        registry_.emplace<KitingAI>(entity, config.kiting_ai.value());
    }
    const auto* transform = registry_.try_get<Transform>(entity);
    if (const auto* collider = registry_.try_get<Collider>(entity); transform && collider) {
        spatial_index_.insert(entity, transform->position, collider->radius);
    }
    return entity;
}

battle::ecs::Entity battle::ecs::World::create_projectile(CreateProjectileConfig config) {
    Entity entity = registry_.create();
    config.context.emitter = entity;
    registry_.emplace<Transform>(entity, config.position, config.direction);
    registry_.emplace<Velocity>(entity, config.direction.x * config.speed, config.direction.y * config.speed);
    registry_.emplace<Projectile>(entity, config.damage, 0.0f, config.max_distance, config.context);

    if (registry_.has<PlayerController>(config.context.owner)) {
        registry_.emplace<Collider>(entity, CollisionShape::Circle, config.hit_radius,
                                    CollisionCategory::PlayerProjectile, PlayerProjectileCollisionMask);
    } else if (registry_.has<MonsterController>(config.context.owner)) {
        registry_.emplace<Collider>(entity, CollisionShape::Circle, config.hit_radius,
                                    CollisionCategory::MonsterProjectile, MonsterProjectileCollisionMask);
    }
    const auto* transform = registry_.try_get<Transform>(entity);
    if (const auto* collider = registry_.try_get<Collider>(entity); transform && collider) {
        spatial_index_.insert(entity, transform->position, collider->radius);
    }
    return entity;
}

battle::ecs::Entity battle::ecs::World::create_obstacle(CreateObstacleConfig config) {
    const Entity entity = registry_.create();
    registry_.emplace<Transform>(entity, config.position, Direction{});
    registry_.emplace<Collider>(entity, CollisionShape::Circle, config.radius, CollisionCategory::Obstacle,
                                ObstacleCollisionMask);
    spatial_index_.insert(entity, config.position, config.radius);
    return entity;
}

battle::ecs::Entity battle::ecs::World::create_trap(CreateTrapConfig config) {
    const Entity entity = registry_.create();
    registry_.emplace<Transform>(entity, config.position, Direction{});
    registry_.emplace<Trap>(entity, config.kind);
    registry_.emplace<Collider>(entity, CollisionShape::Circle, config.radius,
                                CollisionCategory::Trap, TrapCollisionMask);
    spatial_index_.insert(entity, config.position, config.radius);
    return entity;
}

bool battle::ecs::World::has_entity(Entity entity) const {
    return registry_.valid(entity);
}


bool battle::ecs::World::set_player_command(Entity entity, PlayerCommand command) {
    if (!registry_.has<PlayerController>(entity)) {
        return false;
    }
    const bool move_set = set_move_request_(entity, command.move_x, command.move_y);
    const bool attack_set = set_attack_request_(entity, command.attack_requested);
    const bool dash_set = set_dash_request_(entity, command.dash_requested);
    return move_set && attack_set && dash_set;
}

std::shared_ptr<battle::ecs::CombatActionState> battle::ecs::World::create_combat_action() {
    auto state = std::make_shared<CombatActionState>();
    state->action_id = next_combat_action_id_++;
    return state;
}

battle::ecs::CombatEffectID battle::ecs::World::create_combat_effect() {
    return next_combat_effect_id_++;
}

bool battle::ecs::World::relocate_character(Entity entity, const Position& position) {
    auto* transform = registry_.try_get<Transform>(entity);
    const auto* collider = registry_.try_get<Collider>(entity);
    if (!transform || !collider || !is_character(*collider)) {
        return false;
    }

    const auto old_position = transform->position;
    if (!spatial_index_.remove(entity)) {
        return false;
    }

    const auto valid_position = resolve_character_spawn_(position, *collider);
    if (!valid_position.has_value()) {
        spatial_index_.insert(entity, old_position, collider->radius);
        return false;
    }

    transform->position = *valid_position;
    spatial_index_.insert(entity, *valid_position, collider->radius);
    return true;
}

bool battle::ecs::World::set_move_request_(Entity entity, float x, float y) {
    auto* move_request = registry_.try_get<MoveRequest>(entity);
    if (!move_request) {
        return false;
    }
    move_request->x = x;
    move_request->y = y;
    return true;
}

bool battle::ecs::World::set_attack_request_(Entity entity, bool requested) {
    auto* attack_request = registry_.try_get<AttackRequest>(entity);
    if (!attack_request) {
        return false;
    }
    attack_request->requested = requested;
    return true;
}

bool battle::ecs::World::set_dash_request_(Entity entity, bool requested) {
    auto* dash_request = registry_.try_get<DashRequest>(entity);
    if (!dash_request) {
        return false;
    }
    dash_request->requested = requested;
    return true;
}

bool battle::ecs::World::can_place_character_(Position position, const Collider& collider) const {
    if (collider.radius <= 0.0f || bounds_.min_x + collider.radius > bounds_.max_x - collider.radius ||
        bounds_.min_y + collider.radius > bounds_.max_y - collider.radius) {
        return false;
    }
    if (!is_character(collider)) {
        return false;
    }
    if (position.x - collider.radius < bounds_.min_x || position.x + collider.radius > bounds_.max_x ||
        position.y - collider.radius < bounds_.min_y || position.y + collider.radius > bounds_.max_y) {
        return false;
    }
    auto candidates = spatial_index_.query_circle(position, collider.radius);
    return std::ranges::all_of(candidates, [this, &collider, position](const Entity candidate) {
        const auto* transform = registry_.try_get<Transform>(candidate);
        const auto* candidate_collider = registry_.try_get<Collider>(candidate);
        if (!transform || !candidate_collider || !is_interactable(*candidate_collider, collider)) {
            return true;
        }
        return !is_overlap(position, transform->position, collider.radius, candidate_collider->radius);
    });
}

std::optional<battle::ecs::Position> battle::ecs::World::resolve_character_spawn_(Position position,
    const Collider& collider) const {
    if (collider.radius <= 0.0f || bounds_.min_x + collider.radius > bounds_.max_x - collider.radius ||
        bounds_.min_y + collider.radius > bounds_.max_y - collider.radius) {
        return std::nullopt;
    }
    normalize_in_boundary(position, bounds_, collider);
    if (can_place_character_(position, collider)) {
        return position;
    }

    const float step = 2.0f * collider.radius;
    const float max_span = std::max(bounds_.max_x - bounds_.min_x, bounds_.max_y - bounds_.min_y);
    const int max_ring = static_cast<int>(std::ceil(max_span / step));
    for (int ring = 1; ring <= max_ring; ring++) {
        for (int i = 0; i < 4; ++i) {
            for (int j = 0; j < 2 * ring; ++j) {
                int dx{}, dy{};
                switch (i) { //NOLINT
                case 0: {
                    dx = -ring + j;
                    dy = ring;
                    break;
                }
                case 1: {
                    dx = ring;
                    dy = ring - j;
                    break;
                }
                case 2: {
                    dx = ring - j;
                    dy = -ring;
                    break;
                }
                case 3: {
                    dx = -ring;
                    dy = -ring + j;
                    break;
                }
                }
                Position possible_position{
                    .x = position.x + static_cast<float>(dx) * step,
                    .y = position.y + static_cast<float>(dy) * step,
                };
                normalize_in_boundary(possible_position, bounds_, collider);
                if (can_place_character_(possible_position, collider)) {
                    return possible_position;
                }
            }
        }
    }
    return std::nullopt;
}

void battle::ecs::World::tick(DeltaTime delta_time) {
    system_scheduler_.tick(*this, delta_time);
}

void battle::ecs::World::add_kill_event(KillEvent event) {
    kill_events_.emplace_back(event);
}

void battle::ecs::World::add_damage_event(DamageEvent event) {
    damage_events_.emplace_back(std::move(event));
}

void battle::ecs::World::add_damage_applied_event(DamageAppliedEvent event) {
    damage_applied_events().emplace_back(std::move(event));
}

void battle::ecs::World::add_attack_event(AttackEvent event) {
    attack_events_.emplace_back(event);
}

void battle::ecs::World::add_death_event(DeathEvent event) {
    death_events_.emplace_back(event);
}

battle::ecs::WorldSnapshot battle::ecs::World::snapshot() const {
    WorldSnapshot snap_shot;

    for (const auto entity : registry_.entities()) {
        const auto* transform = registry_.try_get<Transform>(entity);
        const auto* health = registry_.try_get<Health>(entity);
        const auto* collider = registry_.try_get<Collider>(entity);
        auto kind = EntityKind::Unknown;
        std::optional<MonsterKind> monster_kind = std::nullopt;
        std::string scene_object_kind;

        if (registry_.has<PlayerController>(entity)) {
            kind = EntityKind::Player;
        } else if (registry_.has<MonsterController>(entity)) {
            auto monster_identity = registry_.try_get<MonsterIdentity>(entity);
            if (!monster_identity) {
                continue;
            }
            kind = EntityKind::Monster;
            monster_kind = std::make_optional(monster_identity->kind);
        } else if (registry_.has<Projectile>(entity)) {
            kind = EntityKind::Projectile;
        } else if (collider != nullptr && collider->category == CollisionCategory::Obstacle) {
            kind = EntityKind::Obstacle;
            scene_object_kind = "obstacle";
        } else if (const auto* trap = registry().try_get<Trap>(entity); trap != nullptr && collider != nullptr &&
            collider->category == CollisionCategory::Trap) {
            kind = EntityKind::Trap;
            switch (trap->kind) {
            case TrapKind::Spikes: {
                scene_object_kind = "spikes";
                break;
            }
            case TrapKind::PoisonPool: {
                scene_object_kind = "poison_pool";
                break;
            }
            case TrapKind::Swamp: {
                scene_object_kind = "swamp";
                break;
            }
            }
        }
        if (transform == nullptr || kind == EntityKind::Unknown) {
            continue;
        }
        snap_shot.entities.emplace_back(entity, kind, transform->position, transform->direction,
                                        health != nullptr ? health->current_health : 0,
                                        health != nullptr ? health->max_health : 0, monster_kind,
                                        collider != nullptr ? collider->radius : 0.0f, std::move(scene_object_kind));
    }
    return snap_shot;
}

bool battle::ecs::World::destroy_entity(Entity entity) {
    spatial_index_.remove(entity);
    return registry_.destroy(entity);
}
