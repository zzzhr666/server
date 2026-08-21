#include "room_graph_presets.hpp"

#include "gameplay_config.hpp"

battle::DungeonRoomGraph battle::default_dungeon_room_graph() {
    return DungeonRoomGraph{
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
                .layout_id = "combat_small",
                .next_room_ids = {3},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = gameplay_config::room::CombatMeleeMonsterCount,
                        },
                        RoomMonsterGroup{
                            .kind = MonsterKind::Ranged,
                            .count = gameplay_config::room::CombatRangedMonsterCount,
                        },
                    },
                },
            },
            DungeonRoomNode{
                .room_id = 3,
                .kind = DungeonRoomKind::Reward,
                .layout_id = "reward_small",
                .next_room_ids = {4},
            },
            DungeonRoomNode{
                .room_id = 4,
                .kind = DungeonRoomKind::Boss,
                .layout_id = "boss_small",
                .next_room_ids = {},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = gameplay_config::room::BossMeleeMonsterCount,
                        },
                    },
                },
            },
        }
    };
}
