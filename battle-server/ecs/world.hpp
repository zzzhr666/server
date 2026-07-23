#pragma once

#include <unordered_map>

#include "component_pool.hpp"
#include "entity.hpp"
#include "components.hpp"
#include "entity_manager.hpp"


namespace battle::ecs {
    struct CreatePlayerConfig {
        std::int64_t player_id;
        float x_position;
        float y_position;
        int max_health;
        float move_speed;
    };

    struct CreateMonsterConfig {
        float x_position;
        float y_position;
        int max_health;
        float move_speed;
    };

    struct EntitySnapshot {
        Entity entity;
        float x_position;
        float y_position;
        float x_direction;
        float y_direction;
        int current_health;
        int max_health;
    };

    struct WorldSnapshot {
        std::vector<EntitySnapshot> entities;
    };

    class World {
    public:
        World();

        Entity create_player(CreatePlayerConfig config);

        Entity create_monster(CreateMonsterConfig config);

        bool has_entity(Entity entity) const;

        bool set_move_input(std::int64_t player_id, float x, float y);

        void tick(float delta_seconds);

        const ComponentPool<Transform>& transforms() const {
            return transforms_;
        }

        const ComponentPool<PlayerController>& player_controllers() const {
            return player_controllers_;
        }

        WorldSnapshot snapshot() const;

        bool destroy_entity(Entity entity);

    private:
        EntityManager entity_manager_;
        ComponentPool<Transform> transforms_;
        ComponentPool<Velocity> velocities_;
        ComponentPool<Health> health_;
        ComponentPool<MoveInput> move_input_;
        ComponentPool<PlayerController> player_controllers_;
        ComponentPool<CharacterStats> character_stats_;

        std::unordered_map<std::int64_t, Entity> player_entities_;
    };
}
