#include "world.hpp"

#include <cmath>

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
#include "system/status_effect_system.hpp"

battle::ecs::World::World(WorldBounds bounds, std::uint32_t random_seed)
    : bounds_(bounds), random_engine_(random_seed), percent_distribution_(1, 100), next_combat_action_id_(1),
      next_combat_effect_id_(1) {
    damage_events_.reserve(InitialDamageEventCount);
    system_scheduler_.add_system(move_resolve_system);
    system_scheduler_.add_system(status_effect_system);
    system_scheduler_.add_system(monster_ai_system);
    system_scheduler_.add_system(dash_resolve_system);
    system_scheduler_.add_system(dash_system);
    system_scheduler_.add_system(move_system);
    system_scheduler_.add_system(attack_resolve_system);
    system_scheduler_.add_system(hit_resolve_system);
    system_scheduler_.add_system(damage_modify_system);
    system_scheduler_.add_system(pre_damage_blessing_system);
    system_scheduler_.add_system(damage_system);
    system_scheduler_.add_system(blessing_trigger_system);
    system_scheduler_.add_system(death_system);
}

battle::ecs::World::World(std::initializer_list<sysFunc> functions, WorldBounds bounds, std::uint32_t random_seed)
    : system_scheduler_(functions), bounds_(bounds), random_engine_(random_seed), percent_distribution_(1, 100),
      next_combat_action_id_(1), next_combat_effect_id_(1) {}

battle::ecs::Entity battle::ecs::World::create_player(CreatePlayerConfig config) {
    Entity entity = registry_.create();

    registry_.emplace<Transform>(entity, Position{.x = config.position.x, .y = config.position.y},
                                 Direction{.x = 0.0f, .y = 1.0f});
    registry_.emplace<Velocity>(entity, 0.0f, 0.0f);
    registry_.emplace<MoveRequest>(entity, 0.0f, 0.0f);

    registry_.emplace<AttackRequest>(entity, false);
    registry_.emplace<DashRequest>(entity, false);
    registry_.emplace<MoveIntent>(entity, 0.0f, 0.0f);
    registry_.emplace<Health>(entity, config.max_health, config.max_health);
    registry_.emplace<CharacterStats>(entity, config.move_speed);
    registry_.emplace<PlayerController>(entity);
    registry_.emplace<AttackIntent>(entity, false, AttackKind::Melee, 0, 0.0f, 0.0f);
    registry_.emplace<Dash>(entity, DefaultPlayerDashCooldown, DefaultPlayerDashSpeedMultiplier);
    registry_.emplace<DashIntent>(entity, false, 0.0f);
    registry_.emplace<AttackDefinition>(entity, config.attack.kind, config.attack.damage, config.attack.range,
                                        config.attack.cooldown_seconds, config.attack.projectile_speed);
    registry_.emplace<AttackCooldown>(entity, DeltaTime{0.0f});
    registry_.emplace<DashCooldown>(entity, DeltaTime{0.0f});
    registry_.emplace<PlayerProgress>(entity, 1, 0, 100, 0);
    registry_.emplace<BlessingInventory>(entity);
    registry_.emplace<StatusEffects>(entity);
    return entity;
}

battle::ecs::Entity battle::ecs::World::create_monster(CreateMonsterConfig config) {
    Entity entity = registry_.create();
    registry_.emplace<Transform>(entity, Position{.x = config.x_position, .y = config.y_position},
                                 Direction{.x = 0.0f, .y = 1.0f});
    registry_.emplace<Velocity>(entity, 0.0f, 0.0f);
    registry_.emplace<Health>(entity, config.max_health, config.max_health);
    registry_.emplace<CharacterStats>(entity, config.move_speed);
    registry_.emplace<MonsterController>(entity);
    registry_.emplace<AttackRequest>(entity, false);
    registry_.emplace<AttackIntent>(entity, false, AttackKind::Melee, 0, 0.0f, 0.0f);
    registry_.emplace<AttackDefinition>(entity, config.attack.kind, config.attack.damage, config.attack.range,
                                        config.attack.cooldown_seconds, config.attack.projectile_speed);
    registry_.emplace<AttackCooldown>(entity, DeltaTime{0.0f});
    registry_.emplace<MonsterIdentity>(entity, config.kind);
    registry_.emplace<StatusEffects>(entity);
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

std::shared_ptr<battle::ecs::CombatActionState> battle::ecs::World::crete_combat_action() {
    auto state = std::make_shared<CombatActionState>();
    state->action_id = next_combat_action_id_++;
    return state;
}

battle::ecs::CombatEffectID battle::ecs::World::create_combat_effect() {
    return next_combat_effect_id_++;
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


void battle::ecs::World::tick(DeltaTime delta_time) {
    system_scheduler_.tick(*this, delta_time);
}

void battle::ecs::World::add_kill_event(KillEvent event) {
    kill_events_.emplace_back(event);
}

void battle::ecs::World::add_damage_event(DamageEvent event) {
    damage_events_.emplace_back(event);
}

void battle::ecs::World::add_damage_applied_event(DamageAppliedEvent event) {
    damage_applied_events().emplace_back(event);
}

battle::ecs::WorldSnapshot battle::ecs::World::snapshot() const {
    WorldSnapshot snap_shot;
    for (const auto entity : registry_.entities()) {
        auto* transform = registry_.try_get<Transform>(entity);
        auto* health = registry_.try_get<Health>(entity);
        auto kind = EntityKind::Unknown;
        if (registry_.has<PlayerController>(entity)) {
            kind = EntityKind::Player;
        } else if (registry_.has<MonsterController>(entity)) {
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
    return registry_.destroy(entity);
}
