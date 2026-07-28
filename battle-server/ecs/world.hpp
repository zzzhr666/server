#pragma once

#include "component/component_pool.hpp"
#include "entity/entity.hpp"
#include "component/components.hpp"
#include "entity/entity_manager.hpp"
#include "system/system_scheduler.hpp"
#include "time.hpp"


namespace battle::ecs {
    constexpr std::size_t InitialDamageEventCount = 256;

    struct CreatePlayerConfig {
        Position position{.x = 0.0f, .y = 0.0f};
        int max_health = 100;
        float move_speed = 5.0f;
        AttackDefinition attack{
            .kind = AttackKind::Melee,
            .damage = 25,
            .range = 1.5f,
            .cooldown_seconds = DeltaTime{0.5f},
            .projectile_speed = 0.0f,
        };
    };

    struct CreateMonsterConfig {
        float x_position;
        float y_position;
        int max_health;
        float move_speed;
        AttackDefinition attack{
            .kind = AttackKind::Melee,
            .damage = 10,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        };
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

        [[nodiscard]] bool has_entity(Entity entity) const;

        [[nodiscard]] bool has_living_players() const {
            return !player_controllers_.entities().empty();
        }

        [[nodiscard]] bool has_living_monsters() const {
            return !monster_controllers_.entities().empty();
        }

        bool set_player_command(Entity entity, PlayerCommand command);

        void tick(DeltaTime delta_time);

        [[nodiscard]] const ComponentPool<Transform>& transforms() const {
            return transforms_;
        }

        ComponentPool<Transform>& transforms() {
            return transforms_;
        }

        [[nodiscard]] const ComponentPool<Velocity>& velocities() const {
            return velocities_;
        }

        ComponentPool<Velocity>& velocities() {
            return velocities_;
        }

        [[nodiscard]] const ComponentPool<Health>& health() const {
            return health_;
        }

        ComponentPool<Health>& health() {
            return health_;
        }

        [[nodiscard]] const ComponentPool<PlayerController>& player_controllers() const {
            return player_controllers_;
        }

        [[nodiscard]] const ComponentPool<MonsterController>& monster_controllers() const {
            return monster_controllers_;
        }

        ComponentPool<MonsterController>& monster_controllers() {
            return monster_controllers_;
        }

        [[nodiscard]] const ComponentPool<MoveRequest>& move_requests() const {
            return move_requests_;
        }

        ComponentPool<MoveRequest>& move_requests() {
            return move_requests_;
        }

        [[nodiscard]] const ComponentPool<AttackRequest>& attack_requests() const {
            return attack_requests_;
        }

        ComponentPool<AttackRequest>& attack_requests() {
            return attack_requests_;
        }

        [[nodiscard]] const ComponentPool<DashRequest>& dash_requests() const {
            return dash_requests_;
        }

        ComponentPool<DashRequest>& dash_requests() {
            return dash_requests_;
        }

        [[nodiscard]] const ComponentPool<MoveIntent>& move_intents() const {
            return move_intents_;
        }

        ComponentPool<MoveIntent>& move_intents() {
            return move_intents_;
        }

        [[nodiscard]] const ComponentPool<CharacterStats>& character_stats() const {
            return character_stats_;
        }

        [[nodiscard]] const ComponentPool<AttackDefinition>& attack_definitions() const {
            return attack_definitions_;
        }

        ComponentPool<AttackDefinition>& attack_definitions() {
            return attack_definitions_;
        }

        [[nodiscard]] const ComponentPool<Dash>& dashes() const {
            return dashes_;
        }

        [[nodiscard]] const ComponentPool<AttackIntent>& attack_intents() const {
            return attack_intents_;
        }

        ComponentPool<AttackIntent>& attack_intents() {
            return attack_intents_;
        }

        ComponentPool<DashIntent>& dash_intents() {
            return dash_intents_;
        }

        ComponentPool<DashCooldown>& dash_cooldowns() {
            return dash_cooldowns_;
        }

        [[nodiscard]] const ComponentPool<DashCooldown>& dash_cooldowns() const {
            return dash_cooldowns_;
        }


        [[nodiscard]] const ComponentPool<AttackCooldown>& attack_cooldowns() const {
            return attack_cooldowns_;
        }

        ComponentPool<AttackCooldown>& attack_cooldowns() {
            return attack_cooldowns_;
        }

        std::vector<DamageEvent>& damage_events() {
            return damage_events_;
        }

        void add_damage_event(DamageEvent event);

        [[nodiscard]] WorldSnapshot snapshot() const;

        bool destroy_entity(Entity entity);

    private:
        bool set_move_request_(Entity entity, float x, float y);

        bool set_attack_request_(Entity entity, bool requested);

        bool set_dash_request_(Entity entity, bool requested);

        EntityManager entity_manager_;
        ComponentPool<Transform> transforms_;
        ComponentPool<Velocity> velocities_;
        ComponentPool<Health> health_;
        ComponentPool<MoveRequest> move_requests_;
        ComponentPool<AttackRequest> attack_requests_;
        ComponentPool<DashRequest> dash_requests_;
        ComponentPool<MoveIntent> move_intents_;
        ComponentPool<PlayerController> player_controllers_;
        ComponentPool<CharacterStats> character_stats_;
        ComponentPool<MonsterController> monster_controllers_;
        ComponentPool<AttackIntent> attack_intents_;
        ComponentPool<DashIntent> dash_intents_;
        ComponentPool<AttackDefinition> attack_definitions_;
        ComponentPool<Dash> dashes_;
        ComponentPool<AttackCooldown> attack_cooldowns_;
        ComponentPool<DashCooldown> dash_cooldowns_;
        std::vector<DamageEvent> damage_events_;
        SystemScheduler system_scheduler_;
    };
}
