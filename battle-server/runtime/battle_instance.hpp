#pragma once

#include <memory>
#include <optional>
#include <random>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

#include "battle_instance_types.hpp"
#include "player_input.hpp"
#include "ecs/world.hpp"
#include "gameplay/spawn_planner.hpp"
#include "gameplay/monster_kind.hpp"
#include "gameplay/room_runtime.hpp"

namespace battle {
    /// @brief BattleInstance 持有单局 World、房间流程、经验、祝福选择和战斗结算状态。
    class BattleInstance {
    public:
        /// @brief 创建并进入初始房间；配置无法进入初始房间时返回 nullptr。
        [[nodiscard]] static std::unique_ptr<BattleInstance> create(BattleInstanceConfig config);

        BattleInstance(const BattleInstance&) = delete;
        BattleInstance& operator=(const BattleInstance&) = delete;
        BattleInstance(BattleInstance&&) = delete;
        BattleInstance& operator=(BattleInstance&&) = delete;

        /// @brief 根据当前阶段推进战斗或奖励选择，并维护结束状态。
        void tick(ecs::DeltaTime delta_time);

        /// @brief 校验玩家归属后将输入写入其 ECS 实体。
        bool receive_input(std::int64_t player_id, PlayerInput input);

        /// @brief 组合 World、房间、进度和短期战斗事件为网络快照。
        [[nodiscard]] BattleWorldSnapshot snapshot() const;

        [[nodiscard]] BattleState state() const {
            return state_;
        }

        [[nodiscard]] BattleEndReason end_reason() const {
            return end_reason_;
        }

        [[nodiscard]] bool ended() const {
            return state_ == BattleState::Ended;
        }

        [[nodiscard]] const std::unordered_map<std::int64_t, PlayerBattleStats>& player_battle_stats() const {
            return player_battle_stats_;
        }

        /// @brief 返回当前累积的玩家击杀统计和结束原因。
        [[nodiscard]] BattleSettlement settlement() const;

        /// @brief 返回当前房间生命周期阶段。
        [[nodiscard]] RoomFlowState room_state() const {
            return room_runtime_.state();
        }

        /// @brief 返回当前房间图节点 ID。
        [[nodiscard]] DungeonRoomID current_room_id() const {
            return room_runtime_.current_room_id();
        }

        [[nodiscard]] ecs::DeltaTime reward_selection_remaining() const {
            return reward_selection_.remaining_seconds;
        }

        [[nodiscard]] std::optional<ecs::PlayerProgress> player_progress(std::int64_t player_id) const;

        [[nodiscard]] std::optional<PlayerBlessingState> player_blessing_state(std::int64_t player_id) const;

        /// @brief 在奖励选择阶段为玩家应用指定候选祝福。
        bool choose_blessing(std::int64_t player_id, int option_id);

        bool select_room_exit(std::int64_t player_id, DungeonRoomID next_room_id);

        void update_connected_players(const std::unordered_set<std::int64_t>& player_ids) {
            connected_player_ids_ = player_ids;
        }


        bool choose_free_reward(std::int64_t player_id, FreeRewardKind kind);

        bool purchase_shop_item(std::int64_t player_id, std::uint32_t item_id);

    private:
        explicit BattleInstance(BattleInstanceConfig config);

        /// @brief 保留至指定 server tick 的表现事件，降低 UDP 丢包的影响。
        struct PendingBattleEvent {
            BattleEvent event;
            std::uint64_t expire_tick{};
        };

        /// @brief 攻击和死亡事件在快照中重复保留的 tick 数。
        static constexpr std::uint64_t EventHistoryTicks = 60;

        /// @brief 将 World 的攻击和死亡事件复制到可重发的事件历史。
        void collect_combat_events_();

        void discard_expired_combat_events_();
        /// @brief 将战斗转为终止状态并记录权威结束原因。
        void end_battle_(BattleEndReason reason);

        /// @brief 消费击杀事件，归属统计并授予玩家经验。
        void consume_kill_events_();

        void tick_blessing_selection_(ecs::DeltaTime delta_time);

        void tick_fighting_(ecs::DeltaTime delta_time);


        /// @brief 为有待选次数的玩家生成候选，并进入奖励选择阶段。
        void start_reward_selection_();

        void apply_default_upgrade_choices_();


        void grant_experience_(std::int64_t player_id, int experience);

        [[nodiscard]] int experience_for_monster_kind_(MonsterKind kind) const;

        [[nodiscard]] int experience_to_next_level_(int level) const;

        std::vector<BlessingOption> generate_blessing_options_(std::int64_t player_id);

        void add_or_level_up_blessing_(ecs::Entity player_entity, PlayerBlessingState& blessing_state,
                                       BlessingID blessing_id);

        [[nodiscard]] bool all_reward_choices_completed_() const;

        [[nodiscard]] bool all_free_reward_choices_completed_() const;

        /// @brief 将阶段时长向上换算为权威模拟 tick 数，零或负时长不占用 tick。
        [[nodiscard]] std::uint64_t duration_to_ticks_(ecs::DeltaTime duration) const;

        /// @brief 进入当前房间，并将规划出的怪物配置实例化到 World。
        bool enter_current_room_();

        /// @brief 根据 World 的存活怪物数将已清空房间推进到出口选择阶段。
        bool update_room_completion_();

        bool relocate_players_for_current_room_(std::vector<std::pair<ecs::Entity, ecs::Position>>& relocated_players);

        bool handle_selection_(FreeRewardKind kind, std::int64_t player_id);

        [[nodiscard]] bool is_current_reward_room_() const;

        bool apply_shop_item_(ecs::Entity entity, const ShopItemDefinition& definition);
    private:
        std::string room_name_;
        ecs::World world_;
        SpawnPlanner spawn_planner_;
        /// @brief 玩家 ID 到 ECS 实体的映射，用于输入和经验归属。
        std::unordered_map<std::int64_t, ecs::Entity> player_entities_;
        std::unordered_map<ecs::Entity, std::int64_t> entity_players_;
        /// @brief 每位玩家开局时冻结的英雄，用于向所有客户端同步角色外观。
        std::unordered_map<std::int64_t, HeroKind> player_heroes_;
        std::unordered_map<std::int64_t, PlayerBattleStats> player_battle_stats_;
        /// @brief 单局生命周期状态和最终结束原因。
        BattleState state_;
        BattleEndReason end_reason_;
        /// @brief Fighting 与 RewardSelection 两阶段的当前状态。
        RewardSelectionState reward_selection_;
        ProgressionConfig progression_config_;
        /// @brief 每位玩家已持有祝福和当前可选候选。
        std::unordered_map<std::int64_t, PlayerBlessingState> player_blessings_;
        std::mt19937 reward_random_engine_;

        /// @brief 快照时间轴使用的每秒权威模拟 tick 数。
        std::uint32_t tick_rate_{DefaultBattleTickRate};
        std::uint64_t server_tick_{};
        std::uint64_t next_event_id_{1};
        std::vector<PendingBattleEvent> pending_battle_events_;

        ecs::DeltaTime combat_elapsed_time_{0.0f};

        /// @brief 单局冻结的房间配置，其生命周期覆盖引用它们的 room_runtime_。
        DungeonRoomGraph dungeon_room_graph_;
        RoomLayoutCatalog room_layout_catalog_;
        RoomRuntime room_runtime_;
        std::unordered_map<std::int64_t, DungeonRoomID> room_exit_choices_;
        /// @brief 当前房间拥有的静态实体；切房时统一销毁并按新布局重建。
        std::vector<ecs::Entity> active_obstacles_;
        std::vector<ecs::Entity> active_traps_;

        bool initialization_failed_{false};

        std::unordered_set<std::int64_t> connected_player_ids_;
        std::unordered_map<std::int64_t, PlayerFreeRewardState> free_reward_states_;
        std::vector<ShopOffer> shop_offers_;
        std::vector<ShopItemDefinition> shop_item_definitions_;

        std::unordered_map<std::int64_t, int> player_souls_;
        std::unordered_map<std::int64_t, std::unordered_set<std::uint32_t>> purchased_shop_items_;
    };
}
