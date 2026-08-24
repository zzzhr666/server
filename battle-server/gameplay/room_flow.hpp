#pragma once

#include <cstdint>
#include <optional>

#include "room_graph.hpp"

namespace battle {
    /// @brief 描述单个房间从进入到切换的流程阶段。
    enum class RoomFlowState : std::uint8_t {
        EnteringRoom,
        Fighting,
        RoomCleared,
        Rewarding,
        ChoosingBlessing,
        ChoosingExit,
        Transitioning,
    };

    class RoomFlow {
    public:
        /// @brief 创建处于指定初始阶段的房间流程。
        explicit RoomFlow(DungeonRoomID room_id,
                          RoomFlowState state = RoomFlowState::EnteringRoom) noexcept;
        /// @brief 返回当前房间流程阶段。
        [[nodiscard]] RoomFlowState state() const noexcept {
            return state_;
        }

        /// @brief 返回流程当前关联的房间 ID。
        [[nodiscard]] DungeonRoomID current_room_id() const noexcept {
            return current_room_id_;
        }

        /// @brief 返回已选择的下一房间；尚未选择时返回空值。
        [[nodiscard]] std::optional<DungeonRoomID> selected_room_id() const noexcept {
            return selected_room_id_;
        }

        /// @brief 尝试切换到下一合法阶段，非法转换时保持当前阶段。
        bool transition_to(RoomFlowState new_state) noexcept;
        /// @brief 在当前房间的合法出口中选择目标并进入切换阶段。
        bool select_exit(DungeonRoomID next_room_id, const DungeonRoomGraph& graph) noexcept;

        /// @brief 完成已选择的切房，将当前房间更新为目标并进入新房间。
        bool complete_transition() noexcept;
    private:
        DungeonRoomID current_room_id_;
        RoomFlowState state_;
        std::optional<DungeonRoomID> selected_room_id_;
    };
}
