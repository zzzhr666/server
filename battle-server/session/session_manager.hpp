#pragma once


#include <memory>
#include <string>
#include <mutex>
#include <unordered_map>
#include <vector>
#include <chrono>

#include "net/udp_endpoint.hpp"

namespace battle {
    class RoomManager;
    class BattleSession;
    class BattleMetrics;

    /// @brief ClientHello 通过校验后创建或重绑会话所需的参数。
    struct JoinSessionRequest {
        std::string room_name;
        std::string token;
        std::int64_t player_id;
        std::uint32_t conv;
        UdpEndpoint endpoint;
    };

    /// @brief 加入或重绑 UDP 会话的领域状态。
    enum class JoinSessionStatus {
        OK = 0,
        InvalidRequest,
        RoomNotFound,
        InvalidToken,
        PlayerNotAllowed,
        AlreadyJoined,
        InternalError,
    };

    /// @brief 会话加入操作的结果，并在成功时携带会话对象。
    struct JoinSessionResult {
        JoinSessionStatus status;
        std::string message;
        bool all_players_joined;
        std::shared_ptr<BattleSession> session;
    };

    /// @brief SessionManager 按玩家、conversation 与房间索引 UDP 会话，并维护三者一致性。
    class SessionManager {
    public:
        SessionManager(RoomManager& room_manager, BattleMetrics& metrics);

        /// @brief 校验房间准入后创建会话，或对已有玩家执行重绑。
        JoinSessionResult join(JoinSessionRequest request);

        /// @brief 返回房间内全部会话，包括已断开的会话。
        std::vector<std::shared_ptr<BattleSession>> sessions_in_room(std::string_view room_name) const;

        /// @brief 关闭并删除房间关联的全部会话索引。
        void remove_room(std::string_view room_name);

        /// @brief 刷新指定玩家会话的活跃时间。
        bool touch(std::string_view room_name, std::int64_t player_id, const UdpEndpoint& endpoint);

        /// @brief 将超过空闲阈值的会话标为断开，并返回本次变化数量。
        std::size_t mark_stale_sessions(std::chrono::steady_clock::time_point now,
                                        std::chrono::steady_clock::duration idle_timeout);

        /// @brief 返回房间内仍处于 Connected 状态的会话。
        std::vector<std::shared_ptr<BattleSession>> connected_sessions_in_room(std::string_view room_name) const;

    private:
        BattleMetrics& metrics_;
        RoomManager& room_manager_;
        mutable std::mutex mutex_;
        /// @brief 以玩家 ID 索引会话，用于同玩家重连。
        std::unordered_map<std::int64_t, std::shared_ptr<BattleSession>> sessions_by_player_;
        std::unordered_map<std::uint32_t, std::shared_ptr<BattleSession>> sessions_by_conv_;
        std::unordered_map<std::string, std::vector<std::shared_ptr<BattleSession>>> sessions_by_room_;
    };
}
