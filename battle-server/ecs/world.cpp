#include "world.hpp"

#include <cmath>

#include "system/input_system.hpp"
#include "system/move_system.hpp"

battle::ecs::World::World() {
    system_scheduler_.add_system(input_system);
    system_scheduler_.add_system(move_system);
}

battle::ecs::World::World(std::initializer_list<sysFunc> functions)
    : system_scheduler_(functions) {}

battle::ecs::Entity battle::ecs::World::create_player(CreatePlayerConfig config) {

    Entity entity = entity_manager_.create();

    transforms_.emplace(entity, Position{.x = config.x_position, .y = config.y_position},
                        Direction{.x = 0.0f, .y = 1.0f});
    velocities_.emplace(entity, 0.0f, 0.0f);
    move_inputs_.emplace(entity, 0.0f, 0.0f);
    health_.emplace(entity, config.max_health, config.max_health);
    character_stats_.emplace(entity, config.move_speed);
    player_controllers_.emplace(entity);

    return entity;
}

battle::ecs::Entity battle::ecs::World::create_monster(CreateMonsterConfig config) {
    Entity entity = entity_manager_.create();
    transforms_.emplace(entity, Position{.x = config.x_position, .y = config.y_position},
                        Direction{.x = 0.0f, .y = 1.0f});
    velocities_.emplace(entity, 0.0f, 0.0f);
    health_.emplace(entity, config.max_health, config.max_health);
    character_stats_.emplace(entity, config.move_speed);

    return entity;
}

bool battle::ecs::World::has_entity(Entity entity) const {
    return entity_manager_.has(entity);
}

bool battle::ecs::World::set_move_input(Entity entity, float x, float y) {

    auto input = move_inputs_.try_get(entity);
    if (!input) {
        return false;
    }
    input->x = x;
    input->y = y;
    return true;
}

void battle::ecs::World::tick(DeltaTime delta_time) {
    system_scheduler_.tick(*this, delta_time);
}

battle::ecs::WorldSnapshot battle::ecs::World::snapshot() const {
    WorldSnapshot snap_shot;
    for (const auto entity : entity_manager_.entities()) {
        auto* transform = transforms_.try_get(entity);
        auto* health = health_.try_get(entity);
        if (transform && health) {
            snap_shot.entities.emplace_back(entity, transform->position.x, transform->position.y,
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
    move_inputs_.remove(entity);
    health_.remove(entity);
    player_controllers_.remove(entity);
    return true;
}
