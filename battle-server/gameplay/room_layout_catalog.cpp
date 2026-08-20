#include "room_layout_catalog.hpp"

#include <algorithm>

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
                    .min_x = -20.0f,
                    .max_x = 20.0f,
                    .min_y = -20.0f,
                    .max_y = 20.0f,
                },
                .player_spawn_points = {
                    ecs::Position{.x = 0.0f, .y = 0.0f},
                },
                .doors = {
                    RoomDoor{
                        .door_id = 1,
                        .position = ecs::Position{.x = 20.0f, .y = 0.0f},
                    },
                },
            },
            RoomLayout{
                .layout_id = "combat_small",
                .bounds = ecs::WorldBounds{
                    .min_x = -20.0f,
                    .max_x = 20.0f,
                    .min_y = -20.0f,
                    .max_y = 20.0f,
                },
                .player_spawn_points = {
                    ecs::Position{.x = -8.0f, .y = 0.0f},
                },
                .monster_spawn_points = {
                    ecs::Position{.x = 8.0f, .y = 0.0f},
                },
                .doors = {
                    RoomDoor{
                        .door_id = 1,
                        .position = ecs::Position{.x = 20.0f, .y = 0.0f},
                    },
                },
            },
            RoomLayout{
                .layout_id = "reward_small",
                .bounds = ecs::WorldBounds{
                    .min_x = -20.0f,
                    .max_x = 20.0f,
                    .min_y = -20.0f,
                    .max_y = 20.0f,
                },
                .player_spawn_points = {
                    ecs::Position{.x = 0.0f, .y = 0.0f},
                },
                .doors = {
                    RoomDoor{
                        .door_id = 1,
                        .position = ecs::Position{.x = 20.0f, .y = 0.0f},
                    },
                },
            },
            RoomLayout{
                .layout_id = "boss_small",
                .bounds = ecs::WorldBounds{
                    .min_x = -20.0f,
                    .max_x = 20.0f,
                    .min_y = -20.0f,
                    .max_y = 20.0f,
                },
                .player_spawn_points = {
                    ecs::Position{.x = -8.0f, .y = 0.0f},
                },
                .monster_spawn_points = {
                    ecs::Position{.x = 8.0f, .y = 0.0f},
                },
            },
        },
    };
}
