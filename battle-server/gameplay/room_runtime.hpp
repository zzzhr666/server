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
        RoomRuntime(const DungeonRoomGraph& dungeon_room_graph, const RoomLayoutCatalog& layout_catalog);

        [[nodiscard]] RoomFlowState state() const noexcept {
            return flow_.state();
        }
        [[nodiscard]] DungeonRoomID current_room_id() const noexcept {
            return flow_.current_room_id();
        }
        [[nodiscard]] const std::vector<ecs::CreateMonsterConfig>& monster_configs() const noexcept {
            return monster_configs_;
        }

        bool prepare_current_room();

        bool start_current_room();

        /// @brief 在战斗阶段根据存活怪物数判断房间是否已清空。
        bool update_living_monster_count(std::size_t living_monster_count);

        /// @brief 在房间清空后进入出口选择阶段。
        bool begin_exit_selection();

        /// @brief 选择当前房间的合法出口并进入房间切换阶段。
        bool select_exit(DungeonRoomID next_room_id);

        /// @brief 提交已选择的房间切换，并丢弃旧房间的怪物生成配置。
        bool complete_transition();

        bool begin_blessing_selection();

    private:
        const DungeonRoomGraph& graph_;
        const RoomLayoutCatalog& catalog_;
        RoomFlow flow_;
        std::vector<ecs::CreateMonsterConfig> monster_configs_;
    };
}
