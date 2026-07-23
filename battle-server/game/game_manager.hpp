#pragma once
#include "room.hpp"

#include <memory>
#include <mutex>
#include <string_view>
#include <unordered_map>

namespace battle {
    /// RoomManager owns active rooms and exposes thread-safe room operations.
    class RoomManager {
    public:
        RoomManager();

        /// Reserves a new room for the matched players.
        CreateRoomResult create_room(CreateRoomRequest request);

        /// Removes a room and releases its reserved player capacity.
        bool close_room(std::string_view room_name);

        /// Checks whether a player can enter a room without mutating join state.
        bool can_join(std::string_view room_name, std::int64_t player_id, std::string_view token) const;

        /// Returns the number of currently reserved rooms.
        std::size_t active_rooms() const;

        /// Returns the number of players reserved across all rooms.
        std::size_t active_players() const;

        /// Marks a player as joined if the room token and whitelist allow it.
        JoinRoomResult join_room(const JoinRoomRequest& request);


    private:
        mutable std::mutex mutex_;
        std::size_t active_players_;
        std::unordered_map<std::string, std::shared_ptr<Room>> rooms_;
    };
}
