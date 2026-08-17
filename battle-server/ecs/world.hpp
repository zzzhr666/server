#pragma once

#include <random>


#include "entity/entity.hpp"
#include "component/components.hpp"
#include "registry.hpp"
#include "system/system_scheduler.hpp"
#include "time.hpp"


namespace battle::ecs {
    /// @brief 伤害事件缓冲区的初始预留容量，避免战斗中频繁扩容。
    constexpr std::size_t InitialDamageEventCount = 256;
    constexpr int DefaultPlayerMaxHealth = 1000;
    constexpr float DefaultPlayerMoveSpeed = 11.0f;
    constexpr int DefaultPlayerAttackDamage = 23;
    constexpr float DefaultPlayerAttackRange = 3.0f;
    constexpr DeltaTime DefaultPlayerAttackCooldown{0.24f};
    constexpr float DefaultPlayerDashSpeedMultiplier = 25.0f;
    constexpr DeltaTime DefaultPlayerDashCooldown{1.0f};

    /// @brief 创建玩家实体时写入的基础组件配置。
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
            .projectile_hit_radius = DefaultProjectileHitRadius,
        };
    };

    /// @brief 创建怪物实体时写入的种类、属性和攻击配置。
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
            .projectile_hit_radius = DefaultProjectileHitRadius,
        };
        std::optional<KitingAI> kiting_ai;
    };

    /// @brief World 中实体可活动的二维边界。
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


    /// @brief 快照中使用的实体类别。
    enum class EntityKind {
        Unknown,
        Player,
        Monster,
        Projectile,
    };

    /// @brief 供运行时序列化的单个实体权威状态。
    struct EntitySnapshot {
        Entity entity;
        EntityKind kind{};
        Position position{};
        Direction direction{};
        int current_health{};
        int max_health{};
        std::optional<MonsterKind> monster_kind;
    };

    /// @brief 单次 tick 后供外部读取的世界快照。
    struct WorldSnapshot {
        std::vector<EntitySnapshot> entities;
    };

    /// @brief 创建投射物实体时写入的运动、伤害和攻击上下文。
    struct CreateProjectileConfig {
        Position position{};
        Direction direction{};
        float speed{};
        int damage{};
        float max_distance{};
        float hit_radius{DefaultProjectileHitRadius};
        CombatContext context;
    };

    /// @brief World 注册的全部组件类型，组件池只能按此集合访问。
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

    /// @brief World 管理 ECS 实体、事件缓冲与固定顺序的战斗系统调度。
    ///
    /// 系统顺序定义玩法因果关系；调整顺序必须同步更新 ECS 回归测试。
    class World {
    public:
        explicit World(WorldBounds bounds = DefaultWorldBounds, std::uint32_t random_seed = std::random_device{}());

        World(std::initializer_list<sysFunc> functions, WorldBounds bounds = DefaultWorldBounds,
              std::uint32_t random_seed = std::random_device{}());

        /// @brief 按配置创建玩家实体及其必需组件。
        Entity create_player(CreatePlayerConfig config);

        /// @brief 按配置创建怪物实体及其必需组件。
        Entity create_monster(CreateMonsterConfig config);

        /// @brief 创建继承攻击上下文的投射物实体。
        Entity create_projectile(CreateProjectileConfig config);

        /// @brief 返回闭区间 [1, 100] 的随机百分比，供可复现的随机玩法使用。
        int random_percent() {
            return percent_distribution_(random_engine_);
        }

        [[nodiscard]] bool has_entity(Entity entity) const;

        /// @brief 将网络输入转换为玩家实体的移动、攻击和冲刺请求。
        bool set_player_command(Entity entity, PlayerCommand command);

        /// @brief 以固定系统顺序推进一次 ECS 世界。
        void tick(DeltaTime delta_time);


        /// @brief 返回待处理伤害事件缓冲区，供系统按阶段消费。
        std::vector<DamageEvent>& damage_events() {
            return damage_events_;
        }

        std::vector<KillEvent>& kill_events() {
            return kill_events_;
        }

        std::vector<DamageAppliedEvent>& damage_applied_events() {
            return damage_applied_events_;
        }

        std::vector<AttackEvent>& attack_events() {
            return attack_events_;
        }

        std::vector<DeathEvent>& death_events() {
            return death_events_;
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

        void clear_attack_events() {
            attack_events_.clear();
        }

        void clear_death_events() {
            death_events_.clear();
        }

        /// @brief 追加击杀事件，供 BattleInstance 统计经验和结算。
        void add_kill_event(KillEvent event);

        void add_damage_event(DamageEvent event);

        void add_damage_applied_event(DamageAppliedEvent event);

        void add_attack_event(AttackEvent event);

        void add_death_event(DeathEvent event);

        /// @brief 收集可同步给客户端的实体快照。
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

        /// @brief 分配一个攻击动作状态，确保同一动作的 proc 不重复触发。
        std::shared_ptr<CombatActionState> crete_combat_action();

        /// @brief 分配一个独立的战斗效果标识。
        CombatEffectID create_combat_effect();

    private:
        bool set_move_request_(Entity entity, float x, float y);

        bool set_attack_request_(Entity entity, bool requested);

        bool set_dash_request_(Entity entity, bool requested);


        WorldRegistry registry_;

        /// @brief 系统间传递的事件缓冲；由运行时在适当时机清理。
        std::vector<KillEvent> kill_events_;
        std::vector<DamageEvent> damage_events_;
        std::vector<DamageAppliedEvent> damage_applied_events_;
        std::vector<AttackEvent> attack_events_;
        std::vector<DeathEvent> death_events_;

        /// @brief 按固定顺序执行玩法系统的调度器。
        SystemScheduler system_scheduler_;
        WorldBounds bounds_;
        std::mt19937 random_engine_;
        std::uniform_int_distribution<int> percent_distribution_;

        /// @brief 单调递增的攻击动作与效果 ID，分别标识 proc 作用域。
        CombatActionID next_combat_action_id_;
        CombatEffectID next_combat_effect_id_;
    };
}
