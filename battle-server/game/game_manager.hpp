#pragma once
#include "room.hpp"

#include <memory>
#include <mutex>
#include <string_view>
#include <unordered_map>

namespace battle {
    class BattleMetrics;

    /// @brief RoomManager 持有活跃房间并提供线程安全的房间操作。
    class RoomManager {
    public:
        /// @brief 使用指标收集器创建房间管理器。
        explicit RoomManager(BattleMetrics& metrics);

        /// @brief 为已匹配玩家预留新房间。
        CreateRoomResult create_room(CreateRoomRequest request);

        /// @brief 删除房间并释放其预留的玩家容量。
        bool close_room(std::string_view room_name);

        /// @brief 校验玩家是否可进入房间，不修改加入状态。
        bool can_join(std::string_view room_name, std::int64_t player_id, std::string_view token) const;

        /// @brief 返回已预留房间冻结的战斗配置。
        std::vector<PlayerLoadout> player_loadouts(std::string_view room_name) const;

        /// @brief 返回当前已预留的房间数。
        std::size_t active_rooms() const;

        /// @brief 返回所有房间合计预留的玩家数。
        std::size_t active_players() const;

        /// @brief 当房间令牌和白名单有效时标记玩家已加入。
        JoinRoomResult join_room(const JoinRoomRequest& request);


    private:
        BattleMetrics& metrics_;
        /// @brief 保护房间表和容量计数。
        mutable std::mutex mutex_;
        /// @brief 所有活跃房间预留的玩家总数。
        std::size_t active_players_;
        std::unordered_map<std::string, std::shared_ptr<Room>> rooms_;
    };
}
