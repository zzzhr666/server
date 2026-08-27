#include "ecs/navigation.hpp"
#include "ecs/world.hpp"

#include <algorithm>

#include <gtest/gtest.h>

namespace battle::ecs {
    namespace {
        MapConfig test_map_config() {
            return MapConfig{
                .bounds = WorldBounds{
                    .min_x = -20.0f,
                    .max_x = 20.0f,
                    .min_y = -20.0f,
                    .max_y = 20.0f,
                },
                .cell_size = 5.0f,
            };
        }

        TEST(NavigationGridTest, FindsPathAroundObstacle) {
            auto config = test_map_config();
            config.obstacles.emplace_back(MapObstacle{
                .center = Position{.x = 0.0f, .y = 0.0f},
                .radius = 6.0f,
            });
            NavigationGrid grid(config);

            const auto path = grid.find_path(Position{.x = -15.0f, .y = 0.0f},
                                             Position{.x = 15.0f, .y = 0.0f}, 1.0f);

            ASSERT_TRUE(path.has_value());
            ASSERT_GT(path->size(), 2U);
            EXPECT_TRUE(std::ranges::all_of(*path, [](const Position position) {
                return distance_squared(position, Position{.x = 0.0f, .y = 0.0f}) > 7.0f * 7.0f;
            }));
        }

        TEST(NavigationGridTest, ReturnsNoPathWhenGoalIsBlocked) {
            auto config = test_map_config();
            config.obstacles.emplace_back(MapObstacle{
                .center = Position{.x = 10.0f, .y = 0.0f},
                .radius = 6.0f,
            });
            NavigationGrid grid(config);

            EXPECT_FALSE(grid.find_path(Position{.x = -15.0f, .y = 0.0f},
                                        Position{.x = 10.0f, .y = 0.0f}, 1.0f).has_value());
        }

        TEST(WorldTest, RebuildNavigationRetainsColliderSpatialIndex) {
            World world(test_map_config(), 7);
            const auto player = world.create_player(CreatePlayerConfig{
                .position = Position{.x = -10.0f, .y = -10.0f},
            });
            ASSERT_NE(player, NullEntity);

            auto rebuilt_config = test_map_config();
            rebuilt_config.bounds = WorldBounds{
                .min_x = -30.0f,
                .max_x = 30.0f,
                .min_y = -30.0f,
                .max_y = 30.0f,
            };
            ASSERT_TRUE(world.rebuild_navigation(rebuilt_config));

            const auto nearby = world.spatial_index().query_circle(Position{.x = -10.0f, .y = -10.0f}, 1.0f);
            EXPECT_NE(std::ranges::find(nearby, player), nearby.end());
            EXPECT_EQ(world.map_config().bounds.max_x, 30.0f);
        }
    } // namespace
} // namespace battle::ecs
