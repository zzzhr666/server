#include "room_encounter_validator.hpp"

#include <unordered_set>

namespace {
    void validate_room_encounter(const battle::DungeonRoomNode& room, std::vector<battle::RoomEncounterIssue>& issues) {
        if (!room.encounter.has_value()) {
            if (room.kind == battle::DungeonRoomKind::Combat) {
                issues.emplace_back(battle::RoomEncounterIssue{
                    .kind = battle::RoomEncounterIssueKind::EncounterMissingForCombatRoom,
                    .room_id = room.room_id,
                });
            } else if (room.kind == battle::DungeonRoomKind::Boss) {
                issues.emplace_back(battle::RoomEncounterIssue{
                    .kind = battle::RoomEncounterIssueKind::EncounterMissingForBossRoom,
                    .room_id = room.room_id,
                });
            }
            return;
        }

        if (room.kind == battle::DungeonRoomKind::Start) {
            issues.emplace_back(battle::RoomEncounterIssue{
                .kind = battle::RoomEncounterIssueKind::EncounterUnexpectedForStartRoom,
                .room_id = room.room_id,
            });
        } else if (room.kind == battle::DungeonRoomKind::Reward) {
            issues.emplace_back(battle::RoomEncounterIssue{
                .kind = battle::RoomEncounterIssueKind::EncounterUnexpectedForRewardRoom,
                .room_id = room.room_id,
            });
        }

        std::unordered_set<battle::MonsterKind> seen_kinds;
        std::unordered_set<battle::MonsterKind> reported_duplicate_kinds;
        for (std::size_t group_index = 0; group_index < room.encounter->monster_groups.size(); ++group_index) {
            const auto& [kind, count] = room.encounter->monster_groups[group_index];
            if (count == 0) {
                issues.emplace_back(battle::RoomEncounterIssue{
                    .kind = battle::RoomEncounterIssueKind::EmptyMonsterGroup,
                    .room_id = room.room_id,
                    .group_index = group_index,
                });
            }
            if (!seen_kinds.insert(kind).second &&
                reported_duplicate_kinds.insert(kind).second) {
                issues.emplace_back(battle::RoomEncounterIssue{
                    .kind = battle::RoomEncounterIssueKind::DuplicateMonsterKindGroup,
                    .room_id = room.room_id,
                    .group_index = group_index,
                });
            }
        }
    }
}

std::vector<battle::RoomEncounterIssue> battle::validate_room_encounters(
    const DungeonRoomGraph& graph) {
    std::vector<RoomEncounterIssue> issues;

    for (const auto& room : graph.rooms) {
        validate_room_encounter(room, issues);
    }

    return issues;
}
