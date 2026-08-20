#include "room_graph_validator.hpp"

#include <queue>
#include <unordered_map>
#include <unordered_set>

#include "room_layout_catalog.hpp"

namespace {
    struct ValidationState {
        bool duplicate_room_id_exists{};
        bool start_room_not_found_exists{};
        bool exit_room_not_found_exists{};
    };

    void validate_duplicate_room_ids(std::vector<battle::DungeonRoomGraphIssue>& issues,
                                     const battle::DungeonRoomGraph& graph, ValidationState& state) {
        std::unordered_set<battle::DungeonRoomID> seen_room_ids;
        std::unordered_set<battle::DungeonRoomID> reported_duplicate_ids;
        for (const auto& room : graph.rooms) {
            if (!seen_room_ids.insert(room.room_id).second && reported_duplicate_ids.insert(room.room_id).second) {
                issues.emplace_back(battle::DungeonRoomGraphIssueKind::DuplicateRoomID,
                                    std::make_optional(room.room_id));
                state.duplicate_room_id_exists = true;
            }
        }
    }

    void validate_start_room(std::vector<battle::DungeonRoomGraphIssue>& issues,
                             const battle::DungeonRoomGraph& graph, ValidationState& state) {
        const auto* node = graph.find_room(graph.start_room_id);
        if (node == nullptr) {
            issues.emplace_back(battle::DungeonRoomGraphIssueKind::StartRoomNotFound,
                                std::make_optional(graph.start_room_id));
            state.start_room_not_found_exists = true;
            return;
        }
        if (node->kind != battle::DungeonRoomKind::Start) {
            issues.emplace_back(battle::DungeonRoomGraphIssueKind::StartRoomKindMismatch,
                                std::make_optional(graph.start_room_id));
        }
    }

    void validate_exit_rooms(std::vector<battle::DungeonRoomGraphIssue>& issues,
                             const battle::DungeonRoomGraph& graph, ValidationState& state) {
        for (const auto& room : graph.rooms) {
            std::unordered_set<battle::DungeonRoomID> seen_exit_ids;
            std::unordered_set<battle::DungeonRoomID> reported_duplicate_exit_ids;
            for (const auto next_room_id : room.next_room_ids) {
                if (!seen_exit_ids.insert(next_room_id).second) {
                    if (reported_duplicate_exit_ids.insert(next_room_id).second) {
                        issues.emplace_back(battle::DungeonRoomGraphIssueKind::DuplicateExit,
                                            std::make_optional(room.room_id),
                                            std::make_optional(next_room_id));
                    }
                    continue;
                }
                if (graph.find_room(next_room_id) == nullptr) {
                    issues.emplace_back(battle::DungeonRoomGraphIssueKind::ExitRoomNotFound,
                                        std::make_optional(room.room_id),
                                        std::make_optional(next_room_id));
                    state.exit_room_not_found_exists = true;
                    continue;
                }
                if (next_room_id == room.room_id) {
                    issues.emplace_back(battle::DungeonRoomGraphIssueKind::SelfLoop,
                                        std::make_optional(room.room_id),
                                        std::make_optional(room.room_id));
                }
            }
        }
    }

    void validate_boss_room(std::vector<battle::DungeonRoomGraphIssue>& issues,
                            const battle::DungeonRoomGraph& graph) {
        bool boss_found = false;
        for (const auto& room : graph.rooms) {
            if (room.kind != battle::DungeonRoomKind::Boss) {
                continue;
            }
            boss_found = true;
            std::unordered_set<battle::DungeonRoomID> seen_exit_ids;
            for (const auto invalid_exit_id : room.next_room_ids) {
                if (seen_exit_ids.insert(invalid_exit_id).second) {
                    issues.emplace_back(battle::DungeonRoomGraphIssueKind::BossHasExit,
                                        std::make_optional(room.room_id),
                                        std::make_optional(invalid_exit_id));
                }
            }
        }
        if (!boss_found) {
            issues.emplace_back(battle::DungeonRoomGraphIssue{
                .kind = battle::DungeonRoomGraphIssueKind::BossRoomNotFound,
            });
        }
    }

    enum class VisitState {
        Unvisited,
        Visiting,
        Visited,
    };

    void validate_cycle_graph(std::vector<battle::DungeonRoomGraphIssue>& issues,
                              const battle::DungeonRoomGraph& graph) {
        std::unordered_map<battle::DungeonRoomID, VisitState> visit_states;
        for (const auto& room : graph.rooms) {
            visit_states.emplace(room.room_id, VisitState::Unvisited);
        }
        auto visit = [&issues, &graph, &visit_states](auto&& self,
                                                      const battle::DungeonRoomID room_id) -> void {
            visit_states.at(room_id) = VisitState::Visiting;
            const auto* room = graph.find_room(room_id);
            std::unordered_set<battle::DungeonRoomID> visited_exit_ids;
            for (const auto next_room_id : room->next_room_ids) {
                if (!visited_exit_ids.insert(next_room_id).second) {
                    continue;
                }
                if (next_room_id == room_id) {
                    continue;
                }
                const auto next_state = visit_states.at(next_room_id);
                if (next_state == VisitState::Visiting) {
                    issues.emplace_back(battle::DungeonRoomGraphIssueKind::CycleDetected,
                                        std::make_optional(room_id), std::make_optional(next_room_id));
                    continue;
                }
                if (next_state == VisitState::Unvisited) {
                    self(self, next_room_id);
                }
            }

            visit_states.at(room_id) = VisitState::Visited;
        };
        for (const auto& room : graph.rooms) {
            if (visit_states.at(room.room_id) == VisitState::Unvisited) {
                visit(visit, room.room_id);
            }
        }
    }

    void validate_reachability(std::vector<battle::DungeonRoomGraphIssue>& issues,
                               const battle::DungeonRoomGraph& graph) {
        std::unordered_set<battle::DungeonRoomID> reachable_ids;
        std::queue<battle::DungeonRoomID> pending_rooms;
        pending_rooms.push(graph.start_room_id);
        while (!pending_rooms.empty()) {
            const auto room_id = pending_rooms.front();
            pending_rooms.pop();
            if (!reachable_ids.insert(room_id).second) {
                continue;
            }
            if (const auto* node = graph.find_room(room_id)) {
                for (const auto next_room_id : node->next_room_ids) {
                    pending_rooms.push(next_room_id);
                }
            }
        }
        for (const auto& room : graph.rooms) {
            if (!reachable_ids.contains(room.room_id)) {
                issues.emplace_back(battle::DungeonRoomGraphIssueKind::RoomUnreachableFromStart,
                                    std::make_optional(room.room_id));
            }
        }
    }
}

std::vector<battle::DungeonRoomGraphIssue> battle::validate_room_graph(const DungeonRoomGraph& graph) {
    ValidationState state;
    std::vector<DungeonRoomGraphIssue> issues;
    validate_duplicate_room_ids(issues, graph, state);
    validate_start_room(issues, graph, state);
    validate_exit_rooms(issues, graph, state);
    if (!state.duplicate_room_id_exists && !state.exit_room_not_found_exists) {
        validate_cycle_graph(issues, graph);
    }
    validate_boss_room(issues, graph);
    if (!state.duplicate_room_id_exists && !state.start_room_not_found_exists &&
        !state.exit_room_not_found_exists) {
        validate_reachability(issues, graph);
    }
    return issues;
}

std::vector<battle::DungeonRoomGraphIssue> battle::validate_room_layout_references(
    const DungeonRoomGraph& graph, const RoomLayoutCatalog& catalog) {
    std::vector<battle::DungeonRoomGraphIssue> issues;
    for (const auto& room : graph.rooms) {
        if (catalog.find_layout(room.layout_id) == nullptr) {
            issues.emplace_back(DungeonRoomGraphIssueKind::LayoutNotFound, std::make_optional(room.room_id));
        }
    }

    return issues;
}
