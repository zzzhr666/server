#include "gameplay/room_graph.hpp"
#include "gameplay/room_graph_presets.hpp"
#include "gameplay/room_graph_validator.hpp"
#include "gameplay/room_flow.hpp"
#include "gameplay/room_encounter_validator.hpp"
#include "gameplay/room_layout.hpp"
#include "gameplay/room_layout_catalog.hpp"
#include "gameplay/room_layout_catalog_validator.hpp"
#include "gameplay/room_layout_validator.hpp"
#include "gameplay/room_runtime.hpp"

#include <gtest/gtest.h>

namespace battle {
namespace {

std::vector<DungeonRoomGraphIssue> issues_of_kind(const std::vector<DungeonRoomGraphIssue>& issues,
                                                   DungeonRoomGraphIssueKind kind) {
    std::vector<DungeonRoomGraphIssue> matching_issues;
    for (const auto& issue : issues) {
        if (issue.kind == kind) {
            matching_issues.emplace_back(issue);
        }
    }
    return matching_issues;
}

TEST(DungeonRoomGraphTest, FindRoomReturnsMatchingNode) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .layout_id = "start",
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .layout_id = "combat",
            },
        },
    };

    const auto* room = graph.find_room(2);

    ASSERT_NE(room, nullptr);
    EXPECT_EQ(room, &graph.rooms[1]);
    EXPECT_EQ(room->kind, DungeonRoomKind::Combat);
    EXPECT_EQ(room->layout_id, "combat");
}

TEST(DungeonRoomGraphTest, FindRoomReturnsNullForUnknownID) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .layout_id = "start",
            },
        },
    };

    EXPECT_EQ(graph.find_room(99), nullptr);
}

TEST(DungeonRoomGraphPresetTest, CreatesValidLinearGraph) {
    const auto graph = default_dungeon_room_graph();

    ASSERT_EQ(graph.start_room_id, 1);
    ASSERT_EQ(graph.rooms.size(), 4);
    EXPECT_EQ(graph.rooms[0].next_room_ids, std::vector<DungeonRoomID>{2});
    EXPECT_EQ(graph.rooms[1].next_room_ids, std::vector<DungeonRoomID>{3});
    EXPECT_EQ(graph.rooms[2].next_room_ids, std::vector<DungeonRoomID>{4});
    EXPECT_TRUE(graph.rooms[3].next_room_ids.empty());
    ASSERT_TRUE(graph.rooms[1].encounter.has_value());
    ASSERT_EQ(graph.rooms[1].encounter->monster_groups.size(), 2);
    ASSERT_TRUE(graph.rooms[3].encounter.has_value());
    ASSERT_EQ(graph.rooms[3].encounter->monster_groups.size(), 1);
    EXPECT_TRUE(validate_room_graph(graph).empty());
}

TEST(RoomEncounterValidatorTest, AcceptsDefaultRoomEncounters) {
    EXPECT_TRUE(validate_room_encounters(default_dungeon_room_graph()).empty());
}

TEST(RoomEncounterValidatorTest, ReportsMissingCombatAndBossEncounters) {
    const DungeonRoomGraph graph{
        .rooms = {
            DungeonRoomNode{.room_id = 2, .kind = DungeonRoomKind::Combat},
            DungeonRoomNode{.room_id = 4, .kind = DungeonRoomKind::Boss},
        },
    };

    const auto issues = validate_room_encounters(graph);

    ASSERT_EQ(issues.size(), 2);
    EXPECT_EQ(issues[0].kind, RoomEncounterIssueKind::EncounterMissingForCombatRoom);
    EXPECT_EQ(issues[1].kind, RoomEncounterIssueKind::EncounterMissingForBossRoom);
}

TEST(RoomEncounterValidatorTest, ReportsUnexpectedStartAndRewardEncounters) {
    const RoomEncounter encounter{
        .monster_groups = {
            RoomMonsterGroup{.kind = MonsterKind::Melee, .count = 1},
        },
    };
    const DungeonRoomGraph graph{
        .rooms = {
            DungeonRoomNode{.room_id = 1, .kind = DungeonRoomKind::Start, .encounter = encounter},
            DungeonRoomNode{.room_id = 3, .kind = DungeonRoomKind::Reward, .encounter = encounter},
        },
    };

    const auto issues = validate_room_encounters(graph);

    ASSERT_EQ(issues.size(), 2);
    EXPECT_EQ(issues[0].kind, RoomEncounterIssueKind::EncounterUnexpectedForStartRoom);
    EXPECT_EQ(issues[1].kind, RoomEncounterIssueKind::EncounterUnexpectedForRewardRoom);
}

TEST(RoomEncounterValidatorTest, ReportsEmptyAndDuplicateMonsterGroups) {
    const DungeonRoomGraph graph{
        .rooms = {
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{.kind = MonsterKind::Melee, .count = 0},
                        RoomMonsterGroup{.kind = MonsterKind::Melee, .count = 2},
                    },
                },
            },
        },
    };

    const auto issues = validate_room_encounters(graph);

    ASSERT_EQ(issues.size(), 2);
    EXPECT_EQ(issues[0].kind, RoomEncounterIssueKind::EmptyMonsterGroup);
    ASSERT_TRUE(issues[0].group_index.has_value());
    EXPECT_EQ(issues[0].group_index.value(), 0);
    EXPECT_EQ(issues[1].kind, RoomEncounterIssueKind::DuplicateMonsterKindGroup);
    ASSERT_TRUE(issues[1].group_index.has_value());
    EXPECT_EQ(issues[1].group_index.value(), 1);
}

TEST(RoomFlowTest, FollowsTheRoomLifecycle) {
    const auto graph = default_dungeon_room_graph();
    RoomFlow flow{3};

    EXPECT_EQ(flow.current_room_id(), 3);
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);
    EXPECT_TRUE(flow.transition_to(RoomFlowState::Fighting));
    EXPECT_TRUE(flow.transition_to(RoomFlowState::RoomCleared));
    EXPECT_TRUE(flow.transition_to(RoomFlowState::ChoosingExit));
    EXPECT_TRUE(flow.select_exit(4, graph));
    EXPECT_TRUE(flow.complete_transition());
    EXPECT_EQ(flow.current_room_id(), 4);
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);
}

TEST(RoomFlowTest, RejectsIllegalTransitionWithoutChangingState) {
    RoomFlow flow{1};

    EXPECT_FALSE(flow.transition_to(RoomFlowState::RoomCleared));
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);
    EXPECT_FALSE(flow.transition_to(RoomFlowState::EnteringRoom));
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);
    EXPECT_FALSE(flow.transition_to(RoomFlowState::Transitioning));
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);
}

TEST(RoomFlowTest, SelectsValidExitWithoutChangingCurrentRoom) {
    const auto graph = default_dungeon_room_graph();
    RoomFlow flow{3, RoomFlowState::ChoosingExit};

    EXPECT_TRUE(flow.select_exit(4, graph));
    EXPECT_EQ(flow.state(), RoomFlowState::Transitioning);
    EXPECT_EQ(flow.current_room_id(), 3);
    ASSERT_TRUE(flow.selected_room_id().has_value());
    EXPECT_EQ(flow.selected_room_id().value(), 4);
}

TEST(RoomFlowTest, RejectsExitThatIsNotInCurrentRoom) {
    const auto graph = default_dungeon_room_graph();
    RoomFlow flow{3, RoomFlowState::ChoosingExit};

    EXPECT_FALSE(flow.select_exit(1, graph));
    EXPECT_EQ(flow.state(), RoomFlowState::ChoosingExit);
    EXPECT_FALSE(flow.selected_room_id().has_value());
}

TEST(RoomFlowTest, RejectsExitSelectionOutsideChoosingExit) {
    const auto graph = default_dungeon_room_graph();
    RoomFlow flow{3};

    EXPECT_FALSE(flow.select_exit(4, graph));
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);
    EXPECT_FALSE(flow.selected_room_id().has_value());
}

TEST(RoomFlowTest, CompletesTransitionAndClearsSelectedRoom) {
    const auto graph = default_dungeon_room_graph();
    RoomFlow flow{3, RoomFlowState::ChoosingExit};

    ASSERT_TRUE(flow.select_exit(4, graph));
    ASSERT_TRUE(flow.selected_room_id().has_value());
    ASSERT_TRUE(flow.complete_transition());

    EXPECT_EQ(flow.current_room_id(), 4);
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);
    EXPECT_FALSE(flow.selected_room_id().has_value());
}

TEST(RoomFlowTest, RejectsCompletingTransitionWithoutTargetOrInWrongState) {
    RoomFlow flow{3};

    EXPECT_FALSE(flow.complete_transition());
    EXPECT_EQ(flow.current_room_id(), 3);
    EXPECT_EQ(flow.state(), RoomFlowState::EnteringRoom);

    RoomFlow transitioning_flow{3, RoomFlowState::Transitioning};
    EXPECT_FALSE(transitioning_flow.complete_transition());
    EXPECT_EQ(transitioning_flow.current_room_id(), 3);
    EXPECT_EQ(transitioning_flow.state(), RoomFlowState::Transitioning);
}

TEST(RoomRuntimeTest, EntersStartRoomWithoutEncounter) {
    const auto graph = default_dungeon_room_graph();
    const auto catalog = default_room_layout_catalog();
    RoomRuntime runtime{graph, catalog};

    EXPECT_EQ(runtime.state(), RoomFlowState::EnteringRoom);
    ASSERT_TRUE(runtime.prepare_current_room());
    EXPECT_TRUE(runtime.start_current_room());
    EXPECT_EQ(runtime.state(), RoomFlowState::Fighting);
    EXPECT_TRUE(runtime.monster_configs().empty());
    EXPECT_FALSE(runtime.start_current_room());
    EXPECT_FALSE(runtime.prepare_current_room());
}

TEST(RoomRuntimeTest, PreparesCombatEncounterBeforeStartingRoom) {
    const DungeonRoomGraph graph{
        .start_room_id = 2,
        .rooms = {
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .layout_id = "combat_small",
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{.kind = MonsterKind::Melee, .count = 2},
                    },
                },
            },
        },
    };
    const auto catalog = default_room_layout_catalog();
    RoomRuntime runtime{graph, catalog};

    ASSERT_TRUE(runtime.prepare_current_room());
    ASSERT_EQ(runtime.monster_configs().size(), 2);
    EXPECT_EQ(runtime.monster_configs()[0].kind, MonsterKind::Melee);
    EXPECT_EQ(runtime.monster_configs()[1].kind, MonsterKind::Melee);
    ASSERT_TRUE(runtime.start_current_room());
    EXPECT_FALSE(runtime.prepare_current_room());
    EXPECT_EQ(runtime.monster_configs().size(), 2);
}

TEST(RoomRuntimeTest, PreparesObstaclesAndTrapsFromRoomLayout) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Combat,
                .layout_id = "hazard_room",
            },
        },
    };
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{
                .layout_id = "hazard_room",
                .obstacles = {
                    RoomObstacle{
                        .obstacle_id = 7,
                        .center = ecs::Position{.x = 2.0f, .y = -3.0f},
                        .radius = 1.5f,
                    },
                },
                .traps = {
                    RoomTrap{
                        .trap_id = 9,
                        .center = ecs::Position{.x = -4.0f, .y = 5.0f},
                        .radius = 2.5f,
                        .kind = TrapKind::PoisonPool,
                    },
                },
            },
        },
    };
    RoomRuntime runtime{graph, catalog};

    ASSERT_TRUE(runtime.prepare_current_room());
    ASSERT_EQ(runtime.obstacle_configs().size(), 1);
    EXPECT_FLOAT_EQ(runtime.obstacle_configs()[0].position.x, 2.0f);
    EXPECT_FLOAT_EQ(runtime.obstacle_configs()[0].position.y, -3.0f);
    EXPECT_FLOAT_EQ(runtime.obstacle_configs()[0].radius, 1.5f);
    ASSERT_EQ(runtime.trap_configs().size(), 1);
    EXPECT_FLOAT_EQ(runtime.trap_configs()[0].position.x, -4.0f);
    EXPECT_FLOAT_EQ(runtime.trap_configs()[0].position.y, 5.0f);
    EXPECT_FLOAT_EQ(runtime.trap_configs()[0].radius, 2.5f);
    EXPECT_EQ(runtime.trap_configs()[0].kind, TrapKind::PoisonPool);
}

TEST(RoomRuntimeTest, KeepsStateWhenCurrentLayoutIsMissing) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .layout_id = "missing",
            },
        },
    };
    const auto catalog = default_room_layout_catalog();
    RoomRuntime runtime{graph, catalog};

    EXPECT_FALSE(runtime.prepare_current_room());
    EXPECT_EQ(runtime.state(), RoomFlowState::EnteringRoom);
    EXPECT_TRUE(runtime.monster_configs().empty());
}

TEST(RoomRuntimeTest, ClearsRoomWhenLastLivingMonsterIsGone) {
    const auto graph = default_dungeon_room_graph();
    const auto catalog = default_room_layout_catalog();
    RoomRuntime runtime{graph, catalog};

    ASSERT_TRUE(runtime.prepare_current_room());
    ASSERT_TRUE(runtime.start_current_room());
    EXPECT_FALSE(runtime.update_living_monster_count(1));
    EXPECT_EQ(runtime.state(), RoomFlowState::Fighting);

    EXPECT_TRUE(runtime.update_living_monster_count(0));
    EXPECT_EQ(runtime.state(), RoomFlowState::RoomCleared);
    EXPECT_FALSE(runtime.update_living_monster_count(0));
    EXPECT_EQ(runtime.state(), RoomFlowState::RoomCleared);

    EXPECT_TRUE(runtime.begin_exit_selection());
    EXPECT_EQ(runtime.state(), RoomFlowState::ChoosingExit);
    EXPECT_FALSE(runtime.begin_exit_selection());
    EXPECT_EQ(runtime.state(), RoomFlowState::ChoosingExit);

    EXPECT_TRUE(runtime.select_exit(2));
    EXPECT_EQ(runtime.state(), RoomFlowState::Transitioning);
    EXPECT_EQ(runtime.current_room_id(), 1);
}

TEST(RoomRuntimeTest, RejectsInvalidExitWithoutChangingRoom) {
    const auto graph = default_dungeon_room_graph();
    const auto catalog = default_room_layout_catalog();
    RoomRuntime runtime{graph, catalog};

    EXPECT_FALSE(runtime.select_exit(2));
    EXPECT_EQ(runtime.state(), RoomFlowState::EnteringRoom);

    ASSERT_TRUE(runtime.prepare_current_room());
    ASSERT_TRUE(runtime.start_current_room());
    ASSERT_TRUE(runtime.update_living_monster_count(0));
    ASSERT_TRUE(runtime.begin_exit_selection());
    EXPECT_FALSE(runtime.select_exit(4));
    EXPECT_EQ(runtime.state(), RoomFlowState::ChoosingExit);
    EXPECT_EQ(runtime.current_room_id(), 1);
}

TEST(RoomRuntimeTest, CompletesTransitionAndEntersNextRoom) {
    const DungeonRoomGraph graph{
        .start_room_id = 2,
        .rooms = {
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .layout_id = "combat_small",
                .next_room_ids = {3},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{.kind = MonsterKind::Melee, .count = 2},
                    },
                },
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Reward,
                .layout_id = "reward_small",
            },
        },
    };
    const auto catalog = default_room_layout_catalog();
    RoomRuntime runtime{graph, catalog};

    ASSERT_TRUE(runtime.prepare_current_room());
    ASSERT_TRUE(runtime.start_current_room());
    ASSERT_EQ(runtime.monster_configs().size(), 2);
    EXPECT_FALSE(runtime.complete_transition());
    EXPECT_EQ(runtime.monster_configs().size(), 2);

    ASSERT_TRUE(runtime.update_living_monster_count(0));
    ASSERT_TRUE(runtime.begin_exit_selection());
    ASSERT_TRUE(runtime.select_exit(3));
    EXPECT_TRUE(runtime.complete_transition());
    EXPECT_EQ(runtime.current_room_id(), 3);
    EXPECT_EQ(runtime.state(), RoomFlowState::EnteringRoom);
    EXPECT_TRUE(runtime.monster_configs().empty());
    EXPECT_TRUE(runtime.obstacle_configs().empty());
    EXPECT_TRUE(runtime.trap_configs().empty());

    ASSERT_TRUE(runtime.prepare_current_room());
    EXPECT_TRUE(runtime.start_current_room());
    EXPECT_EQ(runtime.state(), RoomFlowState::Fighting);
    EXPECT_TRUE(runtime.monster_configs().empty());
}

TEST(RoomLayoutTest, StoresBoundsSpawnPointsAndDoors) {
    const RoomLayout layout{
        .layout_id = "combat_small",
        .bounds = ecs::WorldBounds{
            .min_x = -10.0f,
            .max_x = 10.0f,
            .min_y = -8.0f,
            .max_y = 8.0f,
        },
        .player_spawn_points = {
            ecs::Position{.x = -2.0f, .y = 0.0f},
        },
        .monster_spawn_points = {
            ecs::Position{.x = 4.0f, .y = 0.0f},
        },
        .doors = {
            RoomDoor{
                .door_id = 7,
                .position = ecs::Position{.x = 10.0f, .y = 0.0f},
            },
        },
    };

    ASSERT_EQ(layout.layout_id, "combat_small");
    EXPECT_FLOAT_EQ(layout.bounds.max_x, 10.0f);
    ASSERT_EQ(layout.player_spawn_points.size(), 1);
    EXPECT_FLOAT_EQ(layout.player_spawn_points[0].x, -2.0f);
    ASSERT_EQ(layout.doors.size(), 1);
    EXPECT_EQ(layout.doors[0].door_id, 7);
}

TEST(RoomLayoutCatalogTest, FindLayoutReturnsMatchingLayout) {
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{.layout_id = "start"},
            RoomLayout{.layout_id = "combat_small"},
        },
    };

    const auto* layout = catalog.find_layout("combat_small");

    ASSERT_NE(layout, nullptr);
    EXPECT_EQ(layout, &catalog.layouts[1]);
}

TEST(RoomLayoutCatalogTest, FindLayoutReturnsNullForUnknownID) {
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{.layout_id = "start"},
        },
    };

    EXPECT_EQ(catalog.find_layout("missing"), nullptr);
}

TEST(RoomLayoutCatalogTest, DefaultCatalogMatchesDefaultGraph) {
    const auto graph = default_dungeon_room_graph();
    const auto catalog = default_room_layout_catalog();

    ASSERT_EQ(catalog.layouts.size(), 4);
    EXPECT_TRUE(validate_room_layout_references(graph, catalog).empty());
    EXPECT_TRUE(validate_room_layout_catalog(catalog).empty());
    for (const auto& layout : catalog.layouts) {
        EXPECT_TRUE(validate_room_layout(layout).empty()) << layout.layout_id;
    }
}

TEST(RoomLayoutCatalogValidatorTest, AcceptsUniqueLayoutIDs) {
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{.layout_id = "start"},
            RoomLayout{.layout_id = "combat"},
        },
    };

    EXPECT_TRUE(validate_room_layout_catalog(catalog).empty());
}

TEST(RoomLayoutCatalogValidatorTest, ReportsEachDuplicateLayoutIDOnce) {
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{.layout_id = "start"},
            RoomLayout{.layout_id = "combat"},
            RoomLayout{.layout_id = "combat"},
            RoomLayout{.layout_id = "combat"},
            RoomLayout{.layout_id = "start"},
        },
    };

    const auto issues = validate_room_layout_catalog(catalog);

    ASSERT_EQ(issues.size(), 2);
    EXPECT_EQ(issues[0].kind, RoomLayoutCatalogIssueKind::DuplicateLayoutID);
    EXPECT_EQ(issues[0].layout_id, "combat");
    ASSERT_TRUE(issues[0].layout_index.has_value());
    EXPECT_EQ(issues[0].layout_index.value(), 2);
    EXPECT_EQ(issues[1].layout_id, "start");
    ASSERT_TRUE(issues[1].layout_index.has_value());
    EXPECT_EQ(issues[1].layout_index.value(), 4);
}

TEST(DungeonRoomLayoutReferenceValidatorTest, AcceptsExistingLayoutReferences) {
    const DungeonRoomGraph graph{
        .rooms = {
            DungeonRoomNode{.room_id = 1, .layout_id = "start"},
            DungeonRoomNode{.room_id = 2, .layout_id = "combat"},
        },
    };
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{.layout_id = "start"},
            RoomLayout{.layout_id = "combat"},
        },
    };

    EXPECT_TRUE(validate_room_layout_references(graph, catalog).empty());
}

TEST(DungeonRoomLayoutReferenceValidatorTest, ReportsRoomWithMissingLayout) {
    const DungeonRoomGraph graph{
        .rooms = {
            DungeonRoomNode{.room_id = 1, .layout_id = "start"},
            DungeonRoomNode{.room_id = 2, .layout_id = "missing"},
        },
    };
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{.layout_id = "start"},
        },
    };

    const auto issues = validate_room_layout_references(graph, catalog);

    ASSERT_EQ(issues.size(), 1);
    EXPECT_EQ(issues[0].kind, DungeonRoomGraphIssueKind::LayoutNotFound);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 2);
    EXPECT_FALSE(issues[0].target_room_id.has_value());
}

TEST(DungeonRoomLayoutReferenceValidatorTest, AcceptsRoomsSharingOneLayout) {
    const DungeonRoomGraph graph{
        .rooms = {
            DungeonRoomNode{.room_id = 1, .layout_id = "combat"},
            DungeonRoomNode{.room_id = 2, .layout_id = "combat"},
        },
    };
    const RoomLayoutCatalog catalog{
        .layouts = {
            RoomLayout{.layout_id = "combat"},
        },
    };

    EXPECT_TRUE(validate_room_layout_references(graph, catalog).empty());
}

TEST(RoomLayoutValidatorTest, ReportsTheIndexAndCategoryOfOutsideSpawnPoints) {
    const RoomLayout layout{
        .layout_id = "combat_small",
        .bounds = ecs::WorldBounds{
            .min_x = -10.0f,
            .max_x = 10.0f,
            .min_y = -10.0f,
            .max_y = 10.0f,
        },
        .player_spawn_points = {
            ecs::Position{.x = 0.0f, .y = 0.0f},
            ecs::Position{.x = 11.0f, .y = 0.0f},
        },
        .monster_spawn_points = {
            ecs::Position{.x = -11.0f, .y = 0.0f},
        },
    };

    const auto issues = validate_room_layout(layout);

    ASSERT_EQ(issues.size(), 2);
    EXPECT_EQ(issues[0].kind, RoomLayoutIssueKind::PlayerSpawnOutsideBounds);
    ASSERT_TRUE(issues[0].point_index.has_value());
    EXPECT_EQ(issues[0].point_index.value(), 1);
    EXPECT_FALSE(issues[0].door_id.has_value());
    EXPECT_EQ(issues[1].kind, RoomLayoutIssueKind::MonsterSpawnOutsideBounds);
    ASSERT_TRUE(issues[1].point_index.has_value());
    EXPECT_EQ(issues[1].point_index.value(), 0);
}

TEST(RoomLayoutValidatorTest, ReportsEmptyLayoutID) {
    const RoomLayout layout{};

    const auto issues = validate_room_layout(layout);

    ASSERT_EQ(issues.size(), 1);
    EXPECT_EQ(issues[0].kind, RoomLayoutIssueKind::EmptyLayoutID);
    EXPECT_FALSE(issues[0].door_id.has_value());
    EXPECT_FALSE(issues[0].point_index.has_value());
}

TEST(RoomLayoutValidatorTest, ReportsInvalidBounds) {
    const RoomLayout layout{
        .layout_id = "invalid_bounds",
        .bounds = ecs::WorldBounds{
            .min_x = 5.0f,
            .max_x = -5.0f,
            .min_y = -5.0f,
            .max_y = 5.0f,
        },
    };

    const auto issues = validate_room_layout(layout);

    ASSERT_EQ(issues.size(), 1);
    EXPECT_EQ(issues[0].kind, RoomLayoutIssueKind::InvalidBounds);
}

TEST(RoomLayoutValidatorTest, ReportsDuplicateDoorIDOnce) {
    const RoomLayout layout{
        .layout_id = "duplicate_doors",
        .bounds = ecs::WorldBounds{
            .min_x = -10.0f,
            .max_x = 10.0f,
            .min_y = -10.0f,
            .max_y = 10.0f,
        },
        .doors = {
            RoomDoor{.door_id = 7},
            RoomDoor{.door_id = 7},
            RoomDoor{.door_id = 7},
        },
    };

    const auto issues = validate_room_layout(layout);

    ASSERT_EQ(issues.size(), 1);
    EXPECT_EQ(issues[0].kind, RoomLayoutIssueKind::DuplicateDoorID);
    ASSERT_TRUE(issues[0].door_id.has_value());
    EXPECT_EQ(issues[0].door_id.value(), 7);
}

TEST(RoomLayoutValidatorTest, ReportsDoorOutsideBounds) {
    const RoomLayout layout{
        .layout_id = "outside_door",
        .bounds = ecs::WorldBounds{
            .min_x = -10.0f,
            .max_x = 10.0f,
            .min_y = -10.0f,
            .max_y = 10.0f,
        },
        .doors = {
            RoomDoor{
                .door_id = 9,
                .position = ecs::Position{.x = 11.0f, .y = 0.0f},
            },
        },
    };

    const auto issues = validate_room_layout(layout);

    ASSERT_EQ(issues.size(), 1);
    EXPECT_EQ(issues[0].kind, RoomLayoutIssueKind::DoorOutsideBounds);
    ASSERT_TRUE(issues[0].door_id.has_value());
    EXPECT_EQ(issues[0].door_id.value(), 9);
    EXPECT_FALSE(issues[0].point_index.has_value());
}

TEST(DungeonRoomGraphValidatorTest, DoesNotReportDuplicateForUniqueRoomIDs) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{.room_id = 1},
            DungeonRoomNode{.room_id = 2},
            DungeonRoomNode{.room_id = 3},
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::DuplicateRoomID);

    EXPECT_TRUE(issues.empty());
}

TEST(DungeonRoomGraphValidatorTest, ReportsEachDuplicateIDOnceInConfigurationOrder) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{.room_id = 3},
            DungeonRoomNode{.room_id = 1},
            DungeonRoomNode{.room_id = 3},
            DungeonRoomNode{.room_id = 3},
            DungeonRoomNode{.room_id = 1},
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::DuplicateRoomID);

    ASSERT_EQ(issues.size(), 2);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 3);
    ASSERT_TRUE(issues[1].room_id.has_value());
    EXPECT_EQ(issues[1].room_id.value(), 1);
}

TEST(DungeonRoomGraphValidatorTest, ReportsMissingStartRoom) {
    const DungeonRoomGraph graph{
        .start_room_id = 99,
        .rooms = {
            DungeonRoomNode{.room_id = 1},
            DungeonRoomNode{.room_id = 2},
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::StartRoomNotFound);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 99);
}

TEST(DungeonRoomGraphValidatorTest, ReportsStartRoomKindMismatch) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Combat,
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph),
                                       DungeonRoomGraphIssueKind::StartRoomKindMismatch);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 1);
}

TEST(DungeonRoomGraphValidatorTest, ReportsDuplicateIDAndMissingStartRoomTogether) {
    const DungeonRoomGraph graph{
        .start_room_id = 99,
        .rooms = {
            DungeonRoomNode{.room_id = 1},
            DungeonRoomNode{.room_id = 1},
        },
    };

    const auto all_issues = validate_room_graph(graph);
    const auto duplicate_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::DuplicateRoomID);
    const auto start_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::StartRoomNotFound);

    ASSERT_EQ(duplicate_issues.size(), 1);
    ASSERT_TRUE(duplicate_issues[0].room_id.has_value());
    EXPECT_EQ(duplicate_issues[0].room_id.value(), 1);
    ASSERT_EQ(start_issues.size(), 1);
    ASSERT_TRUE(start_issues[0].room_id.has_value());
    EXPECT_EQ(start_issues[0].room_id.value(), 99);
}

TEST(DungeonRoomGraphValidatorTest, DoesNotReportMissingExitForExistingRoom) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::ExitRoomNotFound);

    EXPECT_TRUE(issues.empty());
}

TEST(DungeonRoomGraphValidatorTest, ReportsExitToMissingRoom) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {99},
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::ExitRoomNotFound);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 1);
    ASSERT_TRUE(issues[0].target_room_id.has_value());
    EXPECT_EQ(issues[0].target_room_id.value(), 99);
}

TEST(DungeonRoomGraphValidatorTest, ReportsSelfLoop) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {1},
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::SelfLoop);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 1);
    ASSERT_TRUE(issues[0].target_room_id.has_value());
    EXPECT_EQ(issues[0].target_room_id.value(), 1);
}

TEST(DungeonRoomGraphValidatorTest, ReportsRepeatedSelfLoopOnlyOncePerIssueKind) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {1, 1, 1},
            },
        },
    };

    const auto all_issues = validate_room_graph(graph);
    const auto self_loop_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::SelfLoop);
    const auto duplicate_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::DuplicateExit);

    ASSERT_EQ(self_loop_issues.size(), 1);
    ASSERT_EQ(duplicate_issues.size(), 1);
    ASSERT_TRUE(duplicate_issues[0].room_id.has_value());
    EXPECT_EQ(duplicate_issues[0].room_id.value(), 1);
    ASSERT_TRUE(duplicate_issues[0].target_room_id.has_value());
    EXPECT_EQ(duplicate_issues[0].target_room_id.value(), 1);
}

TEST(DungeonRoomGraphValidatorTest, ReportsRepeatedMissingExitOnlyOncePerIssueKind) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {99, 99, 99},
            },
        },
    };

    const auto all_issues = validate_room_graph(graph);
    const auto missing_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::ExitRoomNotFound);
    const auto duplicate_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::DuplicateExit);

    ASSERT_EQ(missing_issues.size(), 1);
    ASSERT_EQ(duplicate_issues.size(), 1);
    ASSERT_TRUE(duplicate_issues[0].room_id.has_value());
    EXPECT_EQ(duplicate_issues[0].room_id.value(), 1);
    ASSERT_TRUE(duplicate_issues[0].target_room_id.has_value());
    EXPECT_EQ(duplicate_issues[0].target_room_id.value(), 99);
}

TEST(DungeonRoomGraphValidatorTest, DoesNotReportDuplicateWhenDifferentRoomsShareExitTarget) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {3},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .next_room_ids = {3},
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Reward,
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::DuplicateExit);

    EXPECT_TRUE(issues.empty());
}

TEST(DungeonRoomGraphValidatorTest, AcceptsBranchingGraphThatMerges) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2, 3},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .next_room_ids = {4},
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Reward,
                .next_room_ids = {4},
            },
            DungeonRoomNode{
                .room_id = 4,
                .kind = DungeonRoomKind::Boss,
            },
        },
    };

    EXPECT_TRUE(validate_room_graph(graph).empty());
}

TEST(DungeonRoomGraphValidatorTest, ReportsTwoRoomCycle) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .next_room_ids = {1},
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::CycleDetected);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 2);
    ASSERT_TRUE(issues[0].target_room_id.has_value());
    EXPECT_EQ(issues[0].target_room_id.value(), 1);
}

TEST(DungeonRoomGraphValidatorTest, ReportsLongerRoomCycle) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .next_room_ids = {3},
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Combat,
                .next_room_ids = {1},
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::CycleDetected);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 3);
    ASSERT_TRUE(issues[0].target_room_id.has_value());
    EXPECT_EQ(issues[0].target_room_id.value(), 1);
}

TEST(DungeonRoomGraphValidatorTest, ReportsCycleDisconnectedFromStart) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Combat,
                .next_room_ids = {3},
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Reward,
                .next_room_ids = {2},
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::CycleDetected);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 3);
    ASSERT_TRUE(issues[0].target_room_id.has_value());
    EXPECT_EQ(issues[0].target_room_id.value(), 2);
}

TEST(DungeonRoomGraphValidatorTest, ReportsRoomUnreachableFromStart) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Boss,
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Combat,
                .next_room_ids = {2},
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph),
                                       DungeonRoomGraphIssueKind::RoomUnreachableFromStart);

    ASSERT_EQ(issues.size(), 1);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 3);
    EXPECT_FALSE(issues[0].target_room_id.has_value());
}

TEST(DungeonRoomGraphValidatorTest, AcceptsBossWithoutExit) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Boss,
            },
        },
    };

    EXPECT_TRUE(validate_room_graph(graph).empty());
}

TEST(DungeonRoomGraphValidatorTest, ReportsBossExit) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Boss,
                .next_room_ids = {3},
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Reward,
            },
        },
    };

    const auto issues = validate_room_graph(graph);

    ASSERT_EQ(issues.size(), 1);
    EXPECT_EQ(issues[0].kind, DungeonRoomGraphIssueKind::BossHasExit);
    ASSERT_TRUE(issues[0].room_id.has_value());
    EXPECT_EQ(issues[0].room_id.value(), 2);
    ASSERT_TRUE(issues[0].target_room_id.has_value());
    EXPECT_EQ(issues[0].target_room_id.value(), 3);
}

TEST(DungeonRoomGraphValidatorTest, ReportsRepeatedBossExitOnce) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
                .next_room_ids = {2},
            },
            DungeonRoomNode{
                .room_id = 2,
                .kind = DungeonRoomKind::Boss,
                .next_room_ids = {3, 3, 3},
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Reward,
            },
        },
    };

    const auto issues = validate_room_graph(graph);

    ASSERT_EQ(issues.size(), 2);
    EXPECT_EQ(issues[0].kind, DungeonRoomGraphIssueKind::DuplicateExit);
    EXPECT_EQ(issues[1].kind, DungeonRoomGraphIssueKind::BossHasExit);
    ASSERT_TRUE(issues[1].room_id.has_value());
    EXPECT_EQ(issues[1].room_id.value(), 2);
    ASSERT_TRUE(issues[1].target_room_id.has_value());
    EXPECT_EQ(issues[1].target_room_id.value(), 3);
}

TEST(DungeonRoomGraphValidatorTest, ReportsMissingBossAsGraphLevelIssue) {
    const DungeonRoomGraph graph{
        .start_room_id = 1,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .kind = DungeonRoomKind::Start,
            },
        },
    };

    const auto issues = issues_of_kind(validate_room_graph(graph), DungeonRoomGraphIssueKind::BossRoomNotFound);

    ASSERT_EQ(issues.size(), 1);
    EXPECT_FALSE(issues[0].room_id.has_value());
    EXPECT_FALSE(issues[0].target_room_id.has_value());
}

TEST(DungeonRoomGraphValidatorTest, ReportsExitIssueTogetherWithEarlierValidationIssues) {
    const DungeonRoomGraph graph{
        .start_room_id = 99,
        .rooms = {
            DungeonRoomNode{
                .room_id = 1,
                .next_room_ids = {2},
            },
            DungeonRoomNode{.room_id = 1},
        },
    };

    const auto all_issues = validate_room_graph(graph);
    const auto duplicate_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::DuplicateRoomID);
    const auto start_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::StartRoomNotFound);
    const auto exit_issues = issues_of_kind(all_issues, DungeonRoomGraphIssueKind::ExitRoomNotFound);

    ASSERT_EQ(duplicate_issues.size(), 1);
    ASSERT_EQ(start_issues.size(), 1);
    ASSERT_EQ(exit_issues.size(), 1);
    ASSERT_TRUE(exit_issues[0].room_id.has_value());
    EXPECT_EQ(exit_issues[0].room_id.value(), 1);
    ASSERT_TRUE(exit_issues[0].target_room_id.has_value());
    EXPECT_EQ(exit_issues[0].target_room_id.value(), 2);
}

}  // namespace
}  // namespace battle
