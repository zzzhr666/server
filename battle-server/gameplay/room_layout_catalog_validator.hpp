#pragma once

#include <cstddef>
#include <cstdint>
#include <optional>
#include <string>
#include <vector>

#include "room_layout_catalog.hpp"

namespace battle {
    /// @brief 标识布局目录配置违反的唯一性规则。
    enum class RoomLayoutCatalogIssueKind : std::uint8_t {
        DuplicateLayoutID,
    };

    /// @brief 描述布局目录中的一个配置问题。
    struct RoomLayoutCatalogIssue {
        RoomLayoutCatalogIssueKind kind{};
        std::string layout_id;
        std::optional<std::size_t> layout_index{};
    };

    /// @brief 返回布局目录中的重复布局 ID 问题。
    [[nodiscard]] std::vector<RoomLayoutCatalogIssue> validate_room_layout_catalog(
        const RoomLayoutCatalog& catalog);
}
