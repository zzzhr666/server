#pragma once
#include <cstdint>
#include <mutex>
#include <string>
#include <string_view>
#include <unordered_set>
#include <vector>

namespace battle {
    /// @brief 描述玩家进入战斗时使用的英雄及局外成长等级。
    struct PlayerLoadout {
        /// @brief 配置所属的玩家 ID。
        std::int64_t player_id;
        std::string nickname;
        /// @brief 初始英雄名称。
        std::string hero;
        /// @brief 攻击、攻速、生命和移速的局外等级。
        std::int32_t attack_level;
        std::int32_t attack_speed_level;
        std::int32_t health_level;
        std::int32_t move_speed_level;
    };

    /// @brief rcenter 用于预留战斗房间的请求。
    struct CreateRoomRequest {
        std::string room_name;
        std::string token;
        std::vector<int64_t> player_ids;
        std::vector<PlayerLoadout> player_loadouts;
    };

    /// @brief 房间创建在转换为 protobuf 前使用的领域状态。
    enum class CreateRoomStatus : std::uint8_t {
        OK = 0,
        InvalidRequest,
        AlreadyExists,
        InternalError
    };

    /// @brief 房间领域层处理创建请求后返回的结果。
    struct CreateRoomResult {
        CreateRoomStatus status;
        std::string message;
    };

    /// @brief 玩家证明自己有权进入已预留房间时使用的请求。
    struct JoinRoomRequest {
        std::string room_name;
        std::string token;
        std::int64_t player_id;
    };

    /// @brief 加入房间在转换为 protobuf 前使用的领域状态。
    enum class JoinRoomStatus : std::uint8_t {
        OK = 0,
        InvalidRequest,
        RoomNotFound,
        InvalidToken,
        PlayerNotAllowed,
        AlreadyJoined,
        InternalError
    };

    /// @brief 房间领域层处理加入请求后返回的结果。
    struct JoinRoomResult {
        JoinRoomStatus status;
        std::string message;
        bool all_players_joined;
    };

    /// @brief 外部控制面请求结束房间时使用的参数。
    struct EndRoomRequest {
        std::string room_name;
        std::string reason;
    };

    /// @brief 结束房间的领域处理状态。
    enum class EndRoomStatus : std::uint8_t {
        OK = 0,
        InvalidRequest,
        RoomNotFound,
        InternalError
    };

    /// @brief 结束房间操作的处理结果。
    struct EndRoomResult {
        EndRoomStatus status;
        std::string message;
    };

    /// @brief Room 保存不可变的准入信息以及当前已加入的玩家。
    class Room {
    public:
        /// @brief 根据控制面创建请求冻结房间令牌、玩家名单与负载配置。
        explicit Room(CreateRoomRequest request);

        /// @brief 返回房间的全局唯一名称。
        [[nodiscard]] std::string_view name() const {
            return room_name_;
        }

        /// @brief 返回允许进入本房间的玩家数量。
        [[nodiscard]] std::size_t player_count() const {
            return allowed_player_ids_.size();
        }

        /// @brief 校验玩家 ID 和令牌是否有资格进入，不改变加入状态。
        [[nodiscard]] bool can_join(int64_t player_id, std::string_view token) const;

        /// @brief 返回创建房间时冻结的所有玩家战斗配置。
        [[nodiscard]] std::vector<PlayerLoadout> player_loadouts() const;

        /// @brief 仅在令牌和玩家白名单有效时将玩家加入一次。
        JoinRoomResult join(std::int64_t player_id, std::string_view token);

    private:
        /// @brief 保护准入校验与已加入集合之间的原子性。
        std::mutex join_mutex_;
        std::string room_name_;
        std::string token_;
        std::vector<std::int64_t> allowed_player_ids_;
        std::vector<PlayerLoadout> player_loadouts_;
        /// @brief 已成功加入的玩家，用于防止重复 hello 启动房间。
        std::unordered_set<std::int64_t> joined_player_ids_;
    };
}
