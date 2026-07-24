#pragma once

#include "component/component_pool.hpp"
#include "entity/entity.hpp"
#include "component/components.hpp"
#include "entity/entity_manager.hpp"
#include "system/system_scheduler.hpp"
#include "time.hpp"


namespace battle::ecs {
    struct CreatePlayerConfig {
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

        World(std::initializer_list<sysFunc> functions);

        Entity create_player(CreatePlayerConfig config);

        Entity create_monster(CreateMonsterConfig config);

        bool has_entity(Entity entity) const;

        bool set_move_input(Entity entity, float x, float y);

        void tick(DeltaTime delta_time);

        const ComponentPool<Transform>& transforms() const {
            return transforms_;
        }

        ComponentPool<Transform>& transforms() {
            return transforms_;
        }

        const ComponentPool<Velocity>& velocities() const {
            return velocities_;
        }
        ComponentPool<Velocity>& velocities() {
            return velocities_;
        }

        const ComponentPool<Health>& health() const {
            return health_;
        }
        ComponentPool<Health>& health() {
            return health_;
        }

        const ComponentPool<PlayerController>& player_controllers() const {
            return player_controllers_;
        }

        const ComponentPool<MoveInput>& move_inputs() const {
            return move_inputs_;
        }
        const ComponentPool<CharacterStats>& character_stats() const {
            return character_stats_;
        }


        WorldSnapshot snapshot() const;

        bool destroy_entity(Entity entity);

    private:
        EntityManager entity_manager_;
        ComponentPool<Transform> transforms_;
        ComponentPool<Velocity> velocities_;
        ComponentPool<Health> health_;
        ComponentPool<MoveInput> move_inputs_;
        ComponentPool<PlayerController> player_controllers_;
        ComponentPool<CharacterStats> character_stats_;
        SystemScheduler system_scheduler_;

    };
}
