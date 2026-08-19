#include "room_graph.hpp"

#include <algorithm>

namespace battle {
    const DungeonRoomNode* DungeonRoomGraph::find_room(DungeonRoomID room_id) const {
        const auto it = std::ranges::find_if(rooms, [room_id](const DungeonRoomNode& room) {
            return room.room_id == room_id;
        });
        if (it != rooms.end()) {
            return &*it;
        }
        return nullptr;
    }
}
