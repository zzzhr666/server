#pragma once


#include <array>
#include <optional>
#include <vector>

#include "component/components.hpp"
#include "grid_geometry.hpp"
#include "map_config.hpp"

namespace battle::ecs {

    constexpr float DefaultPassingCost = 1.0f;
    class NavigationGrid {
    public:
        explicit NavigationGrid(MapConfig config);

        [[nodiscard]] bool valid() const noexcept;

        [[nodiscard]] std::optional<std::vector<Position>>
        find_path(Position start, Position goal, float character_radius) const;

    private:
        [[nodiscard]] GridCoordinate world_to_grid_(Position position) const noexcept;

        [[nodiscard]] Position grid_to_world_(GridCoordinate coordinate) const noexcept;


        [[nodiscard]] bool in_bounds_(GridCoordinate coordinate) const noexcept;

        [[nodiscard]] bool walkable_(GridCoordinate coordinate,float character_radius) const noexcept;


        [[nodiscard]] float movement_cost_(GridCoordinate coordinate)const noexcept;

        static std::array<GridCoordinate,4>neighbors_(GridCoordinate coordinate) noexcept;
    private:
        WorldBounds bounds_;
        float cell_size_;
        int width_;
        int height_;
        std::vector<MapObstacle> obstacles_;
        std::vector<MapCostZone> cost_zones_;

    };
}
