#pragma once

#include <cstddef>

#include "component/components.hpp"

namespace battle::ecs {
    constexpr float DefaultGridCellSize = 5.0f;

    [[nodiscard]] float distance_squared(Position lhs, Position rhs) noexcept;

    struct GridCoordinate {
        int x{};
        int y{};

        bool operator==(const GridCoordinate&) const = default;
    };

    struct GridCoordinateHash {
        std::size_t operator()(const GridCoordinate& coordinate) const noexcept;
    };
}
