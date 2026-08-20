#include "room_layout_catalog.hpp"

#include <algorithm>

const battle::RoomLayout* battle::RoomLayoutCatalog::find_layout(std::string_view layout_id) const {
    const auto it = std::ranges::find_if(layouts, [layout_id](const RoomLayout& layout) {
        return layout_id == layout.layout_id;
    });
    return it != layouts.end() ? &*it : nullptr;
}
