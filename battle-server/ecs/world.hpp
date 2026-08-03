#pragma once

#include <random>


#include "entity/entity.hpp"
#include "component/components.hpp"
#include "registry.hpp"
#include "system/system_scheduler.hpp"
#include "time.hpp"


namespace battle::ecs {
    constexpr std::size_t InitialDamageEventCount = 256;
    constexpr int DefaultPlayerMaxHealth = 1000;
    constexpr float DefaultPlayerMoveSpeed = 12.0f;
    constexpr int DefaultPlayerAttackDamage = 25;
    constexpr float DefaultPlayerAttackRange = 3.0f;
    constexpr DeltaTime DefaultPlayerAttackCooldown{0.22f};
    constexpr float DefaultPlayerDashSpeedMultiplier = 10.0f;
    constexpr DeltaTime DefaultPlayerDashCooldown{1.0f};

    struct CreatePlayerConfig {
        Position position{.x = 0.0f, .y = 0.0f};
        int max_health = DefaultPlayerMaxHealth;
        float move_speed = DefaultPlayerMoveSpeed;
        AttackDefinition attack{
            .kind = AttackKind::Melee,
            .damage = DefaultPlayerAttackDamage,
            .range = DefaultPlayerAttackRange,
            .cooldown_seconds = DefaultPlayerAttackCooldown,
            .projectile_speed = 0.0f,
        };
    };

    struct CreateMonsterConfig {
        MonsterKind kind = MonsterKind::Melee;
        float x_position{};
        float y_position{};
        int max_health{};
        float move_speed{};
        AttackDefinition attack{
            .kind = AttackKind::Melee,
            .damage = 10,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        };
        std::optional<KitingAI> kiting_ai;
    };

    struct WorldBounds {
        float min_x;
        float max_x;
        float min_y;
        float max_y;
    };

    constexpr WorldBounds DefaultWorldBounds{
        .min_x = -1000.0f,
        .max_x = 1000.0f,
        .min_y = -1000.0f,
        .max_y = 1000.0f,
    };


    enum class EntityKind {
        Unknown,
        Player,
        Monster,
        Projectile,
    };

    struct EntitySnapshot {
        Entity entity;
        EntityKind kind{};
        Position position{};
        Direction direction{};
        int current_health{};
        int max_health{};
        std::optional<MonsterKind>monster_kind;
    };

    struct WorldSnapshot {
        std::vector<EntitySnapshot> entities;
    };

    struct CreateProjectileConfig {
        Position position{};
        Direction direction{};
        float speed{};
        int damage{};
        float max_distance{};
        CombatContext context;
    };

    using WorldRegistry = Registry<
        Transform,
        Velocity,
        Health,
        MoveRequest,
        AttackRequest,
        DashRequest,
        MoveIntent,
        PlayerController,
        MonsterController,
        CharacterStats,
        AttackIntent,
        DashIntent,
        AttackDefinition,
        Dash,
        AttackCooldown,
        DashCooldown,
        MonsterIdentity,
        PlayerProgress,
        BlessingInventory,
        StatusEffects,
        Projectile,
        KitingAI
    >;

    class World {
    public:
        explicit World(WorldBounds bounds = DefaultWorldBounds, std::uint32_t random_seed = std::random_device{}());

        World(std::initializer_list<sysFunc> functions, WorldBounds bounds = DefaultWorldBounds,
              std::uint32_t random_seed = std::random_device{}());

        Entity create_player(CreatePlayerConfig config);

        Entity create_monster(CreateMonsterConfig config);

        Entity create_projectile(CreateProjectileConfig config);

        int random_percent() {
            return percent_distribution_(random_engine_);
        }

        [[nodiscard]] bool has_entity(Entity entity) const;

        bool set_player_command(Entity entity, PlayerCommand command);

        void tick(DeltaTime delta_time);


        std::vector<DamageEvent>& damage_events() {
            return damage_events_;
        }

        std::vector<KillEvent>& kill_events() {
            return kill_events_;
        }

        std::vector<DamageAppliedEvent>& damage_applied_events() {
            return damage_applied_events_;
        }

        void clear_kill_events() {
            kill_events_.clear();
        }

        void clear_damage_applied_events() {
            damage_applied_events_.clear();
        }

        void clear_damage_events() {
            damage_events_.clear();
        }

        void add_kill_event(KillEvent event);

        void add_damage_event(DamageEvent event);

        void add_damage_applied_event(DamageAppliedEvent event);

        [[nodiscard]] WorldSnapshot snapshot() const;

        bool destroy_entity(Entity entity);

        [[nodiscard]] const WorldBounds& world_bounds() const {
            return bounds_;
        }


        [[nodiscard]] WorldRegistry& registry() noexcept {
            return registry_;
        }

        [[nodiscard]] const WorldRegistry& registry() const noexcept {
            return registry_;
        }

        [[nodiscard]] bool has_living_players() const {
            return !registry_.pool<PlayerController>().empty();
        }

        [[nodiscard]] bool has_living_monsters() const {
            return !registry_.pool<MonsterController>().empty();
        }

        std::shared_ptr<CombatActionState> crete_combat_action();

        CombatEffectID create_combat_effect();

    private:
        bool set_move_request_(Entity entity, float x, float y);

        bool set_attack_request_(Entity entity, bool requested);

        bool set_dash_request_(Entity entity, bool requested);


        WorldRegistry registry_;

        std::vector<KillEvent> kill_events_;
        std::vector<DamageEvent> damage_events_;
        std::vector<DamageAppliedEvent> damage_applied_events_;

        SystemScheduler system_scheduler_;
        WorldBounds bounds_;
        std::mt19937 random_engine_;
        std::uniform_int_distribution<int> percent_distribution_;

        CombatActionID next_combat_action_id_;
        CombatEffectID next_combat_effect_id_;
    };
}
