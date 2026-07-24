#pragma once
#include <cstdint>
#include <mutex>
#include <string>
#include <string_view>
#include <unordered_set>
#include <vector>

namespace battle {
    /// Request used by rcenter to reserve a battle room.
    struct CreateRoomRequest {
        std::string room_name;
        std::string token;
        std::vector<int64_t> player_ids;
    };

    /// Business status for room creation before it is translated to protobuf.
    enum class CreateRoomStatus : std::uint8_t {
        OK = 0,
        InvalidRequest,
        AlreadyExists,
        InternalError
    };

    /// Result returned by the room domain layer after a create request.
    struct CreateRoomResult {
        CreateRoomStatus status;
        std::string message;
    };

    /// Request used when a player proves they may enter a reserved room.
    struct JoinRoomRequest {
        std::string room_name;
        std::string token;
        std::int64_t player_id;
    };

    /// Business status for joining a room before it is translated to protobuf.
    enum class JoinRoomStatus : std::uint8_t {
        OK = 0,
        InvalidRequest,
        RoomNotFound,
        InvalidToken,
        PlayerNotAllowed,
        AlreadyJoined,
        InternalError
    };

    /// Result returned by the room domain layer after a join request.
    struct JoinRoomResult {
        JoinRoomStatus status;
        std::string message;
        bool all_players_joined;
    };

    struct EndRoomRequest {
        std::string room_name;
        std::string reason;
    };

    enum class EndRoomStatus : std::uint8_t {
        OK = 0,
        InvalidRequest,
        RoomNotFound,
        InternalError
    };

    struct EndRoomResult {
        EndRoomStatus status;
        std::string message;
    };

    /// Room stores immutable admission data plus the current joined players.
    class Room {
    public:
        explicit Room(CreateRoomRequest request);

        [[nodiscard]] std::string_view name() const {
            return room_name_;
        }

        [[nodiscard]] std::size_t player_count() const {
            return allowed_player_ids_.size();
        }

        [[nodiscard]] bool can_join(int64_t player_id, std::string_view token) const;

        /// Adds a player once if the token and player whitelist are valid.
        JoinRoomResult join(std::int64_t player_id, std::string_view token);

    private:
        std::mutex join_mutex_;
        std::string room_name_;
        std::string token_;
        std::vector<std::int64_t> allowed_player_ids_;
        std::unordered_set<std::int64_t> joined_player_ids_;
    };
}
