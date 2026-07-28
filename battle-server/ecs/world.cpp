#include "world.hpp"

#include <cmath>

#include "system/attack_resolve_system.hpp"
#include "system/damage_system.hpp"
#include "system/dash_resolve_system.hpp"
#include "system/dash_system.hpp"
#include "system/death_system.hpp"
#include "system/hit_resolve_system.hpp"
#include "system/monster_ai_system.hpp"
#include "system/move_resolve_system.hpp"
#include "system/move_system.hpp"

battle::ecs::World::World(WorldBounds bounds) : bounds_(bounds) {
    damage_events_.reserve(InitialDamageEventCount);
    system_scheduler_.add_system(move_resolve_system);
    system_scheduler_.add_system(monster_ai_system);
    system_scheduler_.add_system(dash_resolve_system);
    system_scheduler_.add_system(dash_system);
    system_scheduler_.add_system(move_system);
    system_scheduler_.add_system(attack_resolve_system);
    system_scheduler_.add_system(hit_resolve_system);
    system_scheduler_.add_system(damage_system);
    system_scheduler_.add_system(death_system);
}

battle::ecs::World::World(std::initializer_list<sysFunc> functions, WorldBounds bounds)
    : system_scheduler_(functions), bounds_(bounds) {}

battle::ecs::Entity battle::ecs::World::create_player(CreatePlayerConfig config) {
    Entity entity = entity_manager_.create();
    transforms_.emplace(entity, Position{.x = config.position.x, .y = config.position.y},
                        Direction{.x = 0.0f, .y = 1.0f});
    velocities_.emplace(entity, 0.0f, 0.0f);
    move_requests_.emplace(entity, 0.0f, 0.0f);
    attack_requests_.emplace(entity, false);
    dash_requests_.emplace(entity, false);
    move_intents_.emplace(entity, 0.0f, 0.0f);
    health_.emplace(entity, config.max_health, config.max_health);
    character_stats_.emplace(entity, config.move_speed);
    player_controllers_.emplace(entity);
    attack_intents_.emplace(entity, false, AttackKind::Melee, 0, 0.0f, 0.0f);
    dashes_.emplace(entity, DefaultPlayerDashCooldown, DefaultPlayerDashSpeedMultiplier);
    dash_intents_.emplace(entity, false, 0.0f);
    attack_definitions_.emplace(entity, config.attack.kind, config.attack.damage, config.attack.range,
                                config.attack.cooldown_seconds, config.attack.projectile_speed);
    attack_cooldowns_.emplace(entity, DeltaTime{0.0f});
    dash_cooldowns_.emplace(entity, DeltaTime{0.0f});
    return entity;
}

battle::ecs::Entity battle::ecs::World::create_monster(CreateMonsterConfig config) {
    Entity entity = entity_manager_.create();
    transforms_.emplace(entity, Position{.x = config.x_position, .y = config.y_position},
                        Direction{.x = 0.0f, .y = 1.0f});
    velocities_.emplace(entity, 0.0f, 0.0f);
    health_.emplace(entity, config.max_health, config.max_health);
    character_stats_.emplace(entity, config.move_speed);
    monster_controllers_.emplace(entity);
    attack_requests_.emplace(entity, false);
    attack_intents_.emplace(entity, false, AttackKind::Melee, 0, 0.0f, 0.0f);
    attack_definitions_.emplace(entity, config.attack.kind, config.attack.damage, config.attack.range,
                                config.attack.cooldown_seconds, config.attack.projectile_speed);
    attack_cooldowns_.emplace(entity, DeltaTime{0.0f});
    monster_identities_.emplace(entity, config.kind);
    return entity;
}

bool battle::ecs::World::has_entity(Entity entity) const {
    return entity_manager_.has(entity);
}


bool battle::ecs::World::set_player_command(Entity entity, PlayerCommand command) {
    if (!player_controllers_.has(entity)) {
        return false;
    }
    const bool move_set = set_move_request_(entity, command.move_x, command.move_y);
    const bool attack_set = set_attack_request_(entity, command.attack_requested);
    const bool dash_set = set_dash_request_(entity, command.dash_requested);
    return move_set && attack_set && dash_set;
}

bool battle::ecs::World::set_move_request_(Entity entity, float x, float y) {
    auto* move_request = move_requests_.try_get(entity);
    if (!move_request) {
        return false;
    }
    move_request->x = x;
    move_request->y = y;
    return true;
}

bool battle::ecs::World::set_attack_request_(Entity entity, bool requested) {
    auto* attack_request = attack_requests_.try_get(entity);
    if (!attack_request) {
        return false;
    }
    attack_request->requested = requested;
    return true;
}

bool battle::ecs::World::set_dash_request_(Entity entity, bool requested) {
    auto* dash_request = dash_requests_.try_get(entity);
    if (!dash_request) {
        return false;
    }
    dash_request->requested = requested;
    return true;
}


void battle::ecs::World::tick(DeltaTime delta_time) {
    system_scheduler_.tick(*this, delta_time);
}

void battle::ecs::World::add_kill_event(KillEvent event) {
    kill_events_.emplace_back(event);
}

void battle::ecs::World::add_damage_event(DamageEvent event) {
    damage_events_.emplace_back(event);
}

battle::ecs::WorldSnapshot battle::ecs::World::snapshot() const {
    WorldSnapshot snap_shot;
    for (const auto entity : entity_manager_.entities()) {
        auto* transform = transforms_.try_get(entity);
        auto* health = health_.try_get(entity);
        auto kind = EntityKind::Unknown;
        if (player_controllers_.has(entity)) {
            kind = EntityKind::Player;
        } else if (monster_controllers_.has(entity)) {
            kind = EntityKind::Monster;
        }
        if (transform && health) {
            snap_shot.entities.emplace_back(entity, kind, transform->position.x, transform->position.y,
                                            transform->direction.x, transform->direction.y,
                                            health->current_health, health->max_health);
        }
    }
    return snap_shot;
}

bool battle::ecs::World::destroy_entity(Entity entity) {
    if (!entity_manager_.destroy(entity)) {
        return false;
    }
    transforms_.remove(entity);
    velocities_.remove(entity);
    character_stats_.remove(entity);
    move_requests_.remove(entity);
    attack_requests_.remove(entity);
    move_intents_.remove(entity);
    health_.remove(entity);
    player_controllers_.remove(entity);
    monster_controllers_.remove(entity);
    attack_cooldowns_.remove(entity);
    attack_intents_.remove(entity);
    attack_definitions_.remove(entity);
    dash_requests_.remove(entity);
    dash_intents_.remove(entity);
    dashes_.remove(entity);
    dash_cooldowns_.remove(entity);
    monster_identities_.remove(entity);
    return true;
}
