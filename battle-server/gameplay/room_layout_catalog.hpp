#pragma once

#include <string_view>
#include <vector>

#include "room_layout.hpp"

namespace battle {
    /// @brief 保存一组可供房间图引用的静态布局配置。
    struct RoomLayoutCatalog {
        std::vector<RoomLayout> layouts;

        /// @brief 按布局 ID 返回只读布局，未找到时返回 nullptr。
        [[nodiscard]] const RoomLayout* find_layout(std::string_view layout_id) const;
    };

    /// @brief 返回与默认房间图匹配的固定布局目录。
    [[nodiscard]] RoomLayoutCatalog default_room_layout_catalog();
}
