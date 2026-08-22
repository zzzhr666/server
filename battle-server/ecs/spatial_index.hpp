#pragma once

#include <cstddef>
#include <unordered_map>
#include <vector>

#include "component/components.hpp"
#include "entity/entity.hpp"
#include "grid_geometry.hpp"

namespace battle::ecs {
    class SpatialIndex {
    public:
        explicit SpatialIndex(float cell_size);

        void insert(Entity entity, Position position, float radius);

        void update(Entity entity, Position position, float radius);

        bool remove(Entity entity);

        std::vector<Entity> query_circle(Position center, float radius) const;

        std::vector<Entity> query_aabb(Position min_corner, Position max_corner) const;

    private:
        struct Cell {
            std::vector<Entity> entities;
        };

        [[nodiscard]] std::vector<GridCoordinate> cells_for_aabb_(Position min_corner,
                                                                   Position max_corner) const;

        float cell_size_;
        std::unordered_map<GridCoordinate, Cell, GridCoordinateHash> cells_;
        std::unordered_map<Entity, std::vector<GridCoordinate>> entity_cells_;
    };
}
