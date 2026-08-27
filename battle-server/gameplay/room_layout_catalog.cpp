#include "room_layout_catalog.hpp"

#include <algorithm>
#include <utility>

#include "gameplay_config.hpp"

const battle::RoomLayout* battle::RoomLayoutCatalog::find_layout(std::string_view layout_id) const {
    const auto it = std::ranges::find_if(layouts, [layout_id](const RoomLayout& layout) {
        return layout_id == layout.layout_id;
    });
    return it != layouts.end() ? &*it : nullptr;
}

namespace {
    battle::RoomLayout make_combat_layout(std::string id, float half_extent,
                                          std::vector<battle::RoomObstacle> obstacles,
                                          std::vector<battle::RoomTrap> traps) {
        return battle::RoomLayout{
            .layout_id = std::move(id),
            .bounds = battle::ecs::WorldBounds{
                .min_x = -half_extent, .max_x = half_extent,
                .min_y = -half_extent, .max_y = half_extent
            },
            .player_spawn_points = {battle::ecs::Position{.x = 0.0f, .y = -half_extent + 2.0f}},
            .monster_spawn_points = {
                {-half_extent + 5.0f, half_extent - 5.0f}, {0.0f, half_extent - 4.0f},
                {half_extent - 5.0f, half_extent - 5.0f}, {-half_extent + 4.0f, 0.0f},
                {half_extent - 4.0f, 0.0f}, {-half_extent + 6.0f, -half_extent + 7.0f},
                {half_extent - 6.0f, -half_extent + 7.0f},
            },
            .doors = {battle::RoomDoor{.door_id = 1, .position = {.x = 0.0f, .y = half_extent - 1.0f}}},
            .obstacles = std::move(obstacles),
            .traps = std::move(traps),
        };
    }

    battle::RoomLayout make_start_layout(float half_extent) {
        return battle::RoomLayout{
            .layout_id = "start",
            .bounds = battle::ecs::WorldBounds{
                .min_x = -half_extent, .max_x = half_extent,
                .min_y = -half_extent, .max_y = half_extent
            },
            .player_spawn_points = {{0.0f, -half_extent + 2.0f}},
            .doors = {battle::RoomDoor{.door_id = 1, .position = {0.0f, half_extent - 1.0f}}},
        };
    }

    battle::RoomLayout make_reward_layout(std::string id, float half_width, float half_height,
                                          float fountain_x, float shop_x) {
        return battle::RoomLayout{
            .layout_id = std::move(id),
            .bounds = battle::ecs::WorldBounds{
                .min_x = -half_width, .max_x = half_width,
                .min_y = -half_height, .max_y = half_height
            },
            .player_spawn_points = {{0.0f, -half_height + 2.0f}},
            .doors = {battle::RoomDoor{.door_id = 1, .position = {0.0f, half_height}}},
            .obstacles = {
                battle::RoomObstacle{1, {fountain_x, 0.0f}, 2.0f, battle::ecs::ObstacleKind::RewardFountain},
                battle::RoomObstacle{2, {shop_x, 0.0f}, 12.0f, battle::ecs::ObstacleKind::Shop},
            },
        };
    }
}

battle::RoomLayoutCatalog battle::default_room_layout_catalog() {
    return RoomLayoutCatalog{
        .layouts = {
            make_start_layout(8.0f),
            make_combat_layout("combat_1", 18.0f, {}, {}),
            make_combat_layout("combat_2", 26.0f,
                               {
                                   battle::RoomObstacle{1, {0.0f, 5.0f}, 6.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{2, {-11.0f, -6.0f}, 3.5f, battle::ecs::ObstacleKind::Generic},
                               },
                               {battle::RoomTrap{1, {0.0f, -7.0f}, 4.0f, battle::TrapKind::Spikes}}),
            make_reward_layout("reward_small", 26.0f, 20.0f, -13.0f, 10.0f),
            make_combat_layout("combat_3", 27.0f,
                               {
                                   battle::RoomObstacle{1, {-9.0f, 5.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{2, {9.0f, -5.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{3, {0.0f, -13.0f}, 3.5f, battle::ecs::ObstacleKind::Generic},
                               },
                               {battle::RoomTrap{1, {0.0f, 0.0f}, 4.0f, battle::TrapKind::Swamp}}),
            make_combat_layout("combat_4", 29.0f,
                               {
                                   battle::RoomObstacle{1, {-9.5f, 6.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{2, {9.5f, -6.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{3, {0.0f, -14.0f}, 3.5f, battle::ecs::ObstacleKind::Generic},
                               },
                               {
                                   battle::RoomTrap{1, {-4.0f, -5.0f}, 4.0f, battle::TrapKind::Spikes},
                                   battle::RoomTrap{2, {5.0f, 8.0f}, 4.0f, battle::TrapKind::PoisonPool},
                               }),
            make_combat_layout("combat_5", 35.0f,
                               {
                                   battle::RoomObstacle{1, {-16.0f, 13.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{2, {16.0f, 13.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{3, {0.0f, -14.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                                   battle::RoomObstacle{4, {0.0f, 18.0f}, 4.0f, battle::ecs::ObstacleKind::Generic},
                               },
                               {
                                   battle::RoomTrap{1, {-8.0f, -4.0f}, 4.0f, battle::TrapKind::Spikes},
                                   battle::RoomTrap{2, {8.0f, -4.0f}, 4.0f, battle::TrapKind::PoisonPool},
                                   battle::RoomTrap{3, {0.0f, 8.0f}, 4.0f, battle::TrapKind::Swamp},
                               }),
            make_reward_layout("reward_large", 30.0f, 24.0f, -15.0f, 12.0f),
            battle::RoomLayout{
                .layout_id = "boss_large",
                .bounds = ecs::WorldBounds{.min_x = -52.0f, .max_x = 52.0f, .min_y = -52.0f, .max_y = 52.0f},
                .player_spawn_points = {{0.0f, -45.0f}},
                .monster_spawn_points = {{0.0f, 26.0f}},
                .doors = {},
                .obstacles = {
                    battle::RoomObstacle{1, {-22.0f, 12.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                    battle::RoomObstacle{2, {22.0f, 12.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                    battle::RoomObstacle{3, {0.0f, -24.0f}, 5.0f, battle::ecs::ObstacleKind::Generic},
                },
                .traps = {
                    battle::RoomTrap{1, {-12.0f, -8.0f}, 4.0f, battle::TrapKind::Spikes},
                    battle::RoomTrap{2, {12.0f, -8.0f}, 4.0f, battle::TrapKind::PoisonPool},
                    battle::RoomTrap{3, {0.0f, 4.0f}, 4.0f, battle::TrapKind::Swamp},
                },
            },
        }
    };
}
