#pragma once

#include <vector>

#include "room_flow.hpp"
#include "ecs/world.hpp"

namespace battle {
    struct DungeonRoomGraph;
    struct RoomLayoutCatalog;

    /// @brief 持有当前房间流程，并根据房间图规划本房间的怪物生成配置。
    class RoomRuntime {
    public:
        /// @brief 使用房间图与布局目录创建房间流程运行时。
        RoomRuntime(const DungeonRoomGraph& dungeon_room_graph, const RoomLayoutCatalog& layout_catalog);

        /// @brief 返回当前房间流程阶段。
        [[nodiscard]] RoomFlowState state() const noexcept {
            return flow_.state();
        }
        /// @brief 返回当前房间图节点 ID。
        [[nodiscard]] DungeonRoomID current_room_id() const noexcept {
            return flow_.current_room_id();
        }
        /// @brief 返回当前房间规划出的怪物创建配置。
        [[nodiscard]] const std::vector<ecs::CreateMonsterConfig>& monster_configs() const noexcept {
            return monster_configs_;
        }

        /// @brief 返回当前房间布局中的障碍物创建配置。
        [[nodiscard]] const std::vector<ecs::CreateObstacleConfig>& obstacle_configs() const noexcept {
            return obstacle_configs_;
        }

        /// @brief 返回当前房间布局中的陷阱创建配置。
        [[nodiscard]] const std::vector<ecs::CreateTrapConfig>& trap_configs() const noexcept {
            return trap_configs_;
        }

        /// @brief 根据当前布局生成怪物、障碍物和陷阱创建配置，不改变房间阶段。
        bool prepare_current_room();

        /// @brief 在房间实体成功创建后进入战斗阶段。
        bool start_current_room();

        /// @brief 在战斗阶段根据存活怪物数判断房间是否已清空。
        bool update_living_monster_count(std::size_t living_monster_count);

        /// @brief 在房间清空后进入出口选择阶段。
        bool begin_exit_selection();

        /// @brief 选择当前房间的合法出口并进入房间切换阶段。
        bool select_exit(DungeonRoomID next_room_id);

        /// @brief 提交已选择的房间切换，并丢弃旧房间的全部实体创建配置。
        bool complete_transition();

        /// @brief 在奖励房流程中进入祝福选择阶段。
        bool begin_blessing_selection();

    private:
        const DungeonRoomGraph& graph_;
        const RoomLayoutCatalog& catalog_;
        RoomFlow flow_;
        std::vector<ecs::CreateMonsterConfig> monster_configs_;
        std::vector<ecs::CreateObstacleConfig> obstacle_configs_;
        std::vector<ecs::CreateTrapConfig> trap_configs_;
    };
}
