#include "room_layout_catalog.hpp"

#include <algorithm>

#include "gameplay_config.hpp"

const battle::RoomLayout* battle::RoomLayoutCatalog::find_layout(std::string_view layout_id) const {
    const auto it = std::ranges::find_if(layouts, [layout_id](const RoomLayout& layout) {
        return layout_id == layout.layout_id;
    });
    return it != layouts.end() ? &*it : nullptr;
}

battle::RoomLayoutCatalog battle::default_room_layout_catalog() {
    return RoomLayoutCatalog{
        .layouts = {
            RoomLayout{
                .layout_id = "start",
                .bounds = ecs::WorldBounds{
                    .min_x = gameplay_config::room::MinCoordinate,
                    .max_x = gameplay_config::room::MaxCoordinate,
                    .min_y = gameplay_config::room::MinCoordinate,
                    .max_y = gameplay_config::room::MaxCoordinate,
                },
                .player_spawn_points = {
                    ecs::Position{.x = 0.0f, .y = gameplay_config::room::PlayerSpawnY},
                },
                .doors = {
                    RoomDoor{
                        .door_id = 1,
                        .position = ecs::Position{.x = 0.0f, .y = gameplay_config::room::MaxCoordinate},
                    },
                },
            },
            RoomLayout{
                .layout_id = "combat_small",
                .bounds = ecs::WorldBounds{
                    .min_x = gameplay_config::room::MinCoordinate,
                    .max_x = gameplay_config::room::MaxCoordinate,
                    .min_y = gameplay_config::room::MinCoordinate,
                    .max_y = gameplay_config::room::MaxCoordinate,
                },
                .player_spawn_points = {
                    ecs::Position{.x = 0.0f, .y = gameplay_config::room::PlayerSpawnY},
                },
                .monster_spawn_points = {
                    ecs::Position{.x = -12.0f, .y = 8.0f},
                    ecs::Position{.x = 0.0f, .y = 10.0f},
                    ecs::Position{.x = 12.0f, .y = 8.0f},
                    ecs::Position{.x = -14.0f, .y = 0.0f},
                    ecs::Position{.x = 14.0f, .y = 0.0f},
                    ecs::Position{.x = -10.0f, .y = -8.0f},
                    ecs::Position{.x = 10.0f, .y = -8.0f},
                },
                .doors = {
                    RoomDoor{
                        .door_id = 1,
                        .position = ecs::Position{.x = 0.0f, .y = gameplay_config::room::MaxCoordinate},
                    },
                },
                .obstacles = {
                    RoomObstacle{
                        .obstacle_id = 1,
                        .center = ecs::Position{.x = -5.0f, .y = 1.0f},
                        .radius = 1.4f,
                    },
                    RoomObstacle{
                        .obstacle_id = 2,
                        .center = ecs::Position{.x = 3.0f, .y = 4.0f},
                        .radius = 1.1f,
                    },
                },
                .traps = {
                    RoomTrap{
                        .trap_id = 1,
                        .center = ecs::Position{.x = -2.0f, .y = 3.0f},
                        .radius = 1.0f,
                        .kind = TrapKind::Spikes,
                    },
                    RoomTrap{
                        .trap_id = 2,
                        .center = ecs::Position{.x = 5.0f, .y = -3.0f},
                        .radius = 1.5f,
                        .kind = TrapKind::PoisonPool,
                    },
                    RoomTrap{
                        .trap_id = 3,
                        .center = ecs::Position{.x = -5.0f, .y = -4.0f},
                        .radius = 1.7f,
                        .kind = TrapKind::Swamp,
                    },
                },
            },
            RoomLayout{
                .layout_id = "reward_small",
                .bounds = ecs::WorldBounds{
                    .min_x = gameplay_config::room::MinCoordinate,
                    .max_x = gameplay_config::room::MaxCoordinate,
                    .min_y = gameplay_config::room::MinCoordinate,
                    .max_y = gameplay_config::room::MaxCoordinate,
                },
                .player_spawn_points = {
                    ecs::Position{.x = 0.0f, .y = gameplay_config::room::PlayerSpawnY},
                },
                .doors = {
                    RoomDoor{
                        .door_id = 1,
                        .position = ecs::Position{.x = 0.0f, .y = gameplay_config::room::MaxCoordinate},
                    },
                },
            },
            RoomLayout{
                .layout_id = "boss_small",
                .bounds = ecs::WorldBounds{
                    .min_x = gameplay_config::room::MinCoordinate,
                    .max_x = gameplay_config::room::MaxCoordinate,
                    .min_y = gameplay_config::room::MinCoordinate,
                    .max_y = gameplay_config::room::MaxCoordinate,
                },
                .player_spawn_points = {
                    ecs::Position{.x = 0.0f, .y = gameplay_config::room::PlayerSpawnY},
                },
                .monster_spawn_points = {
                    ecs::Position{.x = 0.0f, .y = 10.0f},
                },
            },
        },
    };
}
