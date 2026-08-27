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
        /// @brief 使用指定网格边长创建空间索引。
        explicit SpatialIndex(float cell_size);

        /// @brief 将实体占据的空间范围加入索引。
        void insert(Entity entity, Position position, float radius);

        /// @brief 更新实体空间范围，并迁移其覆盖的网格单元。
        void update(Entity entity, Position position, float radius);

        /// @brief 从空间索引移除实体。
        bool remove(Entity entity);

        /// @brief 返回可能与指定圆形范围相交的实体。
        std::vector<Entity> query_circle(Position center, float radius) const;

        /// @brief 返回可能与指定轴对齐包围盒相交的实体。
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
