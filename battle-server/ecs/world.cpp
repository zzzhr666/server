#include "world.hpp"

#include <cmath>

battle::ecs::World::World() : entity_manager_() {}

battle::ecs::Entity battle::ecs::World::create_player(CreatePlayerConfig config) {
    if (player_entities_.contains(config.player_id)) {
        return INVALID_ENTITY;
    }

    Entity entity = entity_manager_.create();

    transforms_.emplace(entity, Position{.x = config.x_position, .y = config.y_position},
                        Direction{.x = 0.0f, .y = 1.0f});
    velocities_.emplace(entity, 0.0f, 0.0f);
    move_input_.emplace(entity, 0.0f, 0.0f);
    health_.emplace(entity, config.max_health, config.max_health);
    character_stats_.emplace(entity, config.move_speed);
    player_controllers_.emplace(entity, config.player_id);

    player_entities_[config.player_id] = entity;
    return entity;
}

bool battle::ecs::World::has_entity(Entity entity) const {
    return entity_manager_.has(entity);
}

bool battle::ecs::World::set_move_input(std::int64_t player_id, float x, float y) {
    auto it = player_entities_.find(player_id);
    if (it == player_entities_.end()) {
        return false;
    }
    Entity entity = it->second;
    auto input = move_input_.try_get(entity);
    if (!input) {
        return false;
    }
    input->x = x;
    input->y = y;
    return true;
}

void battle::ecs::World::tick(float delta_seconds) {
    for (auto entity : entity_manager_.entities()) {
        auto* transform = transforms_.try_get(entity);
        auto* velocity = velocities_.try_get(entity);
        auto* move_input = move_input_.try_get(entity);
        auto* character_stat = character_stats_.try_get(entity);
        if (!transform || !velocity || !move_input || !character_stat) {
            continue;
        }
        float len = std::sqrt(move_input->x * move_input->x + move_input->y * move_input->y);
        if (len <= 0.0001f) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            continue;
        }
        float dir_x = move_input->x / len;
        float dir_y = move_input->y / len;
        velocity->x = dir_x * character_stat->move_speed;
        velocity->y = dir_y * character_stat->move_speed;
        transform->position += Position{.x = velocity->x * delta_seconds, .y = velocity->y * delta_seconds};
        transform->direction = {.x = dir_x, .y = dir_y};
    }
}
