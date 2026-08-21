#include "spatial_index.hpp"

#include <algorithm>
#include <cmath>
#include <functional>
#include <unordered_set>
#include <utility>

namespace {
    float normalize_cell_size(float cell_size) {
        if (!std::isfinite(cell_size) || cell_size <= 0) {
            return battle::ecs::DefaultCellSize;
        }
        return cell_size;
    }
}

battle::ecs::SpatialIndex::SpatialIndex(float cell_size) : cell_size_(normalize_cell_size(cell_size)) {}

void battle::ecs::SpatialIndex::insert(Entity entity, Position position, float radius) {
    if (entity_cells_.contains(entity)) {
        update(entity, position, radius);
        return;
    }

    float x_max = position.x + radius;
    float y_max = position.y + radius;
    float x_min = position.x - radius;
    float y_min = position.y - radius;
    auto cell_coordinates = cells_for_aabb_(Position{.x = x_min, .y = y_min}, Position{.x = x_max, .y = y_max});
    for (const auto cell_coordinate : cell_coordinates) {
        auto it = cells_.find(cell_coordinate);
        if (it == cells_.end()) {
            cells_.emplace(cell_coordinate, Cell{.entities = {entity}});
        } else {
            it->second.entities.emplace_back(entity);
        }
    }
    entity_cells_[entity] = std::move(cell_coordinates);
}

void battle::ecs::SpatialIndex::update(Entity entity, Position position, float radius) {
    float x_max = position.x + radius;
    float y_max = position.y + radius;
    float x_min = position.x - radius;
    float y_min = position.y - radius;
    auto cell_coordinates = cells_for_aabb_(Position{.x = x_min, .y = y_min}, Position{.x = x_max, .y = y_max});
    auto entity_it = entity_cells_.find(entity);
    if (entity_it == entity_cells_.end()) {
        insert(entity, position, radius);
        return;
    }
    if (entity_it->second == cell_coordinates) {
        return;
    }
    std::vector<CellCoordinate> possible_empty_cells;
    for (const auto old_coordinate : entity_it->second) {
        auto cell_it = cells_.find(old_coordinate);
        if (cell_it == cells_.end()) {
            continue;
        }
        std::erase(cell_it->second.entities, entity);
        if (cell_it->second.entities.empty()) {
            possible_empty_cells.emplace_back(cell_it->first);
        }
    }
    for (const auto cell_coordinate : cell_coordinates) {
        auto cell_it = cells_.find(cell_coordinate);
        if (cell_it == cells_.end()) {
            cells_.emplace(cell_coordinate, Cell{.entities = {entity}});
        } else {
            cell_it->second.entities.emplace_back(entity);
        }
    }
    entity_it->second = std::move(cell_coordinates);
    for (const auto& c : possible_empty_cells) {
        if (cells_[c].entities.empty()) {
            cells_.erase(c);
        }
    }
}

bool battle::ecs::SpatialIndex::remove(Entity entity) {
    auto it = entity_cells_.find(entity);
    if (it == entity_cells_.end()) {
        return false;
    }
    std::vector<CellCoordinate> empty_cells;
    for (const auto coordinate : it->second) {
        auto cell_it = cells_.find(coordinate);
        if (cell_it == cells_.end()) {
            continue;
        }
        std::erase(cell_it->second.entities, entity);
        if (cell_it->second.entities.empty()) {
            empty_cells.emplace_back(cell_it->first);
        }
    }
    for (const auto& c : empty_cells) {
        cells_.erase(c);
    }
    entity_cells_.erase(it);
    return true;
}

std::vector<battle::ecs::Entity> battle::ecs::SpatialIndex::query_circle(Position center, float radius) const {
    const float x_max = center.x + radius;
    const float y_max = center.y + radius;
    const float x_min = center.x - radius;
    const float y_min = center.y - radius;
    return query_aabb({.x = x_min, .y = y_min}, {.x = x_max, .y = y_max});
}

std::vector<battle::ecs::Entity> battle::ecs::SpatialIndex::query_aabb(Position min_corner, Position max_corner) const {
    auto cell_coordinates = cells_for_aabb_(min_corner, max_corner);
    std::unordered_set<Entity> entities;
    for (const auto& cell_coordinate : cell_coordinates) {
        auto cell_it = cells_.find(cell_coordinate);
        if (cell_it == cells_.end()) {
            continue;
        }
        for (const auto entity : cell_it->second.entities) {
            entities.insert(entity);
        }
    }
    std::vector<Entity> result{entities.begin(), entities.end()};
    std::ranges::sort(result);
    return result;
}

std::size_t battle::ecs::SpatialIndex::CellCoordinateHash::operator()(const CellCoordinate& coordinate) const noexcept {
    const auto packed = (static_cast<std::uint64_t>(static_cast<std::uint32_t>(coordinate.x)) << 32) |
        static_cast<std::uint32_t>(coordinate.y);
    return std::hash<std::uint64_t>{}(packed);
}

std::vector<battle::ecs::SpatialIndex::CellCoordinate> battle::ecs::SpatialIndex::cells_for_aabb_(Position min_corner,
    Position max_corner) const {
    std::vector<CellCoordinate> cells;
    if (min_corner.x > max_corner.x) {
        std::swap(min_corner.x, max_corner.x);
    }
    if (min_corner.y > max_corner.y) {
        std::swap(min_corner.y, max_corner.y);
    }
    const CellCoordinate start{
        .x = static_cast<int>(std::floor(min_corner.x / cell_size_)),
        .y = static_cast<int>(std::floor(min_corner.y / cell_size_)),
    };
    const CellCoordinate end{
        .x = static_cast<int>(std::floor(max_corner.x / cell_size_)),
        .y = static_cast<int>(std::floor(max_corner.y / cell_size_)),
    };
    for (int i = start.x; i <= end.x; i++) {
        for (int j = start.y; j <= end.y; j++) {
            cells.emplace_back(i, j);
        }
    }

    return cells;
}
