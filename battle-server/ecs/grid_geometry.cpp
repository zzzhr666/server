#include "grid_geometry.hpp"

#include <functional>

float battle::ecs::distance_squared(Position lhs, Position rhs) noexcept {
    const float delta_x = lhs.x - rhs.x;
    const float delta_y = lhs.y - rhs.y;
    return delta_x * delta_x + delta_y * delta_y;
}

std::size_t battle::ecs::GridCoordinateHash::operator()(
    const GridCoordinate& coordinate) const noexcept {
    const auto packed =
        (static_cast<std::uint64_t>(static_cast<std::uint32_t>(coordinate.x)) << 32) |
        static_cast<std::uint32_t>(coordinate.y);
    return std::hash<std::uint64_t>{}(packed);
}
