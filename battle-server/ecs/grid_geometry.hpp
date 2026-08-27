#pragma once

#include <cstddef>

#include "component/components.hpp"

namespace battle::ecs {
    constexpr float DefaultGridCellSize = 1.0f;

    /// @brief 返回两个世界坐标间的平方距离。
    [[nodiscard]] float distance_squared(Position lhs, Position rhs) noexcept;

    struct GridCoordinate {
        int x{};
        int y{};

        /// @brief 按网格行列比较坐标。
        bool operator==(const GridCoordinate&) const = default;
    };

    struct GridCoordinateHash {
        /// @brief 为网格坐标计算哈希值。
        std::size_t operator()(const GridCoordinate& coordinate) const noexcept;
    };
}
