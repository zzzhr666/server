#include "navigation.hpp"

#include <algorithm>
#include <cmath>
#include <queue>
#include <unordered_map>
#include <utility>

namespace {
    bool valid_cost_zone(battle::ecs::MapCostZone cost_zone) noexcept {
        const auto& [center,radius,cost_multiplier] = cost_zone;
        return std::isfinite(center.x) && std::isfinite(center.y) && radius > 0 && std::isfinite(radius)
            && std::isfinite(cost_multiplier) && cost_multiplier >= 1.0f;
    }

    constexpr std::array<battle::ecs::GridCoordinate, 4> directions{
        {
            {.x = 1, .y = 0}, {.x = -1, .y = 0}, {.x = 0, .y = 1}, {.x = 0, .y = -1}
        }
    };

    struct OpenNode {
        battle::ecs::GridCoordinate coordinate;
        float g_score{};
        float priority{};
    };

    struct OpenNodeCompare {
        bool operator()(const OpenNode& a, const OpenNode& b) const noexcept {
            return a.priority > b.priority;
        }
    };
}

battle::ecs::MapConfig battle::ecs::DefaultMapConfig() {
    return MapConfig{
        .bounds = DefaultWorldBounds,
        .cell_size = DefaultGridCellSize,
        .obstacles = {},
        .cost_zones = {},
    };
}

battle::ecs::NavigationGrid::NavigationGrid(MapConfig config)
    : bounds_(config.bounds), cell_size_(config.cell_size), width_(0), height_(0),
      obstacles_(std::move(config.obstacles)), cost_zones_(std::move(config.cost_zones)) {
    if (!std::isfinite(cell_size_) || cell_size_ <= 0 ||
        !std::isfinite(bounds_.min_x) || !std::isfinite(bounds_.max_x) ||
        !std::isfinite(bounds_.min_y) || !std::isfinite(bounds_.max_y) ||
        bounds_.min_x >= bounds_.max_x || bounds_.min_y >= bounds_.max_y) {
        return;
    }

    width_ = static_cast<int>(std::ceil((bounds_.max_x - bounds_.min_x) / cell_size_));
    height_ = static_cast<int>(std::ceil((bounds_.max_y - bounds_.min_y) / cell_size_));
}

bool battle::ecs::NavigationGrid::valid() const noexcept {
    return cell_size_ > 0 && std::isfinite(cell_size_) &&
        bounds_.min_x < bounds_.max_x && bounds_.min_y < bounds_.max_y &&
        height_ > 0 && width_ > 0;
}

std::optional<std::vector<battle::ecs::Position>> battle::ecs::NavigationGrid::find_path(Position start, Position goal,
    float character_radius) const {
    if (!valid() || !std::isfinite(start.x) || !std::isfinite(start.y) ||
        !std::isfinite(goal.x) || !std::isfinite(goal.y) ||
        !std::isfinite(character_radius) || character_radius < 0) {
        return std::nullopt;
    }
    const auto start_node = world_to_grid_(start);
    const auto goal_node = world_to_grid_(goal);
    if (!in_bounds_(start_node) || !in_bounds_(goal_node) ||
        !walkable_(start_node, character_radius) || !walkable_(goal_node, character_radius)) {
        return std::nullopt;
    }
    std::unordered_map<GridCoordinate, float, GridCoordinateHash> g_score;
    std::unordered_map<GridCoordinate, GridCoordinate, GridCoordinateHash> came_from;
    std::priority_queue<OpenNode, std::vector<OpenNode>, OpenNodeCompare> open_set;

    const auto heuristic = [goal_node](GridCoordinate coordinate) {
        return static_cast<float>(std::abs(coordinate.x - goal_node.x) + std::abs(coordinate.y - goal_node.y));
    };
    g_score[start_node] = 0;
    open_set.emplace(start_node, 0.0f, heuristic(start_node));

    bool found = false;
    while (!open_set.empty()) {
        auto current = open_set.top();
        open_set.pop();
        const auto current_score = g_score.find(current.coordinate);
        if (current_score == g_score.end() || current.g_score != current_score->second) {
            continue;
        }
        if (current.coordinate == goal_node) {
            found = true;
            break;
        }
        for (auto neighbor : neighbors_(current.coordinate)) {
            if (!in_bounds_(neighbor) || !walkable_(neighbor, character_radius)) {
                continue;
            }
            float tentative_g = g_score[current.coordinate] + movement_cost_(neighbor);
            auto it = g_score.find(neighbor);
            if (it != g_score.end() && it->second <= tentative_g) {
                continue;
            }
            came_from[neighbor] = current.coordinate;
            g_score[neighbor] = tentative_g;
            open_set.emplace(neighbor, tentative_g, tentative_g + heuristic(neighbor));
        }
    }
    if (!found) {
        return std::nullopt;
    }
    std::vector<Position> result{};
    auto current_node = goal_node;
    while (current_node != start_node) {
        auto it = came_from.find(current_node);
        if (it == came_from.end()) {
            return std::nullopt;
        }
        result.emplace_back(grid_to_world_(current_node));
        current_node = it->second;
    }
    result.emplace_back(grid_to_world_(start_node));
    std::ranges::reverse(result);
    return result;
}

battle::ecs::GridCoordinate battle::ecs::NavigationGrid::world_to_grid_(Position position) const noexcept {
    return GridCoordinate{
        .x = static_cast<int>(std::floor((position.x - bounds_.min_x) / cell_size_)),
        .y = static_cast<int>(std::floor((position.y - bounds_.min_y) / cell_size_)),
    };
}

battle::ecs::Position battle::ecs::NavigationGrid::grid_to_world_(GridCoordinate coordinate) const noexcept {
    return Position{
        .x = bounds_.min_x + (static_cast<float>(coordinate.x) + 0.5f) * cell_size_,
        .y = bounds_.min_y + (static_cast<float>(coordinate.y) + 0.5f) * cell_size_,
    };
}

bool battle::ecs::NavigationGrid::in_bounds_(GridCoordinate coordinate) const noexcept {
    return coordinate.x >= 0 && coordinate.x < width_ &&
        coordinate.y >= 0 && coordinate.y < height_;
}

bool battle::ecs::NavigationGrid::walkable_(GridCoordinate coordinate, float character_radius) const noexcept {
    if (!valid() || !in_bounds_(coordinate)) {
        return false;
    }
    if (!std::isfinite(character_radius) || character_radius < 0) {
        return false;
    }
    const auto node_center = grid_to_world_(coordinate);
    return std::ranges::all_of(obstacles_, [node_center, character_radius](const auto& obstacle) {
        if (!std::isfinite(obstacle.center.x) || !std::isfinite(obstacle.center.y) ||
            !std::isfinite(obstacle.radius) || obstacle.radius < 0) {
            return true;
        }
        const float effective_radius = obstacle.radius + character_radius;
        return distance_squared(node_center, obstacle.center) > effective_radius * effective_radius;
    });
}

float battle::ecs::NavigationGrid::movement_cost_(GridCoordinate coordinate) const noexcept {
    auto center_pos = grid_to_world_(coordinate);
    float result = DefaultPassingCost;
    for (const auto& cost_zone : cost_zones_) {
        if (!valid_cost_zone(cost_zone)) {
            continue;
        }
        if (cost_zone.radius * cost_zone.radius > distance_squared(center_pos, cost_zone.center)) {
            result = std::max(result, cost_zone.cost_multiplier * DefaultPassingCost);
        }
    }
    return result;
}

std::array<battle::ecs::GridCoordinate, 4> battle::ecs::NavigationGrid::neighbors_(
    GridCoordinate coordinate) noexcept {
    std::array<GridCoordinate, 4> result{directions};
    for (auto& [x,y] : result) {
        x += coordinate.x;
        y += coordinate.y;
    }
    return result;
}
