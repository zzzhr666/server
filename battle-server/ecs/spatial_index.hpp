#pragma once

#include <cstddef>
#include <unordered_map>
#include <vector>

#include "component/components.hpp"
#include "entity/entity.hpp"

namespace battle::ecs {
    constexpr float DefaultCellSize = 5.0f;
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

        struct CellCoordinate {
            int x;
            int y;
            bool operator==(const CellCoordinate&) const = default;
        };

        struct CellCoordinateHash {
            std::size_t operator()(const CellCoordinate& coordinate) const noexcept;
        };

        [[nodiscard]] std::vector<CellCoordinate> cells_for_aabb_(Position min_corner,
                                                                 Position max_corner) const;

        float cell_size_;
        std::unordered_map<CellCoordinate, Cell, CellCoordinateHash> cells_;
        std::unordered_map<Entity, std::vector<CellCoordinate>> entity_cells_;
    };
}
