#include "gameplay/room_graph.hpp"
#include "gameplay/room_graph_validator.hpp"

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
                .kind = DungeonRoomKind::Elite,
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
