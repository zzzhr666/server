#pragma once

#include <vector>

#include "component/components.hpp"

namespace battle::ecs {
    struct WorldBounds {
        float min_x;
        float max_x;
        float min_y;
        float max_y;
    };

    constexpr WorldBounds DefaultWorldBounds{
        .min_x = -1000.0f,
        .max_x = 1000.0f,
        .min_y = -1000.0f,
        .max_y = 1000.0f,
    };

    struct MapObstacle {
        Position center;
        float radius;
    };

    struct MapCostZone {
        Position center;
        float radius;
        float cost_multiplier;
    };

    struct MapConfig {
        WorldBounds bounds;
        float cell_size;
        std::vector<MapObstacle> obstacles;
        std::vector<MapCostZone> cost_zones;
    };

    /// @brief 返回 battle-server 的默认地图边界与网格配置。
    MapConfig DefaultMapConfig();
}
