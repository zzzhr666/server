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
                .layout_id = "combat_1",
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
                .kind = DungeonRoomKind::Combat,
                .layout_id = "combat_2",
                .next_room_ids = {4},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{MonsterKind::Melee, gameplay_config::room::CombatTierTwoMeleeMonsterCount},
                        RoomMonsterGroup{MonsterKind::Ranged, gameplay_config::room::CombatTierTwoRangedMonsterCount},
                    },
                },
            },
            DungeonRoomNode{
                .room_id = 4,
                .kind = DungeonRoomKind::Reward,
                .layout_id = "reward_small",
                .next_room_ids = {5},
            },
            DungeonRoomNode{
                .room_id = 5,
                .kind = DungeonRoomKind::Combat,
                .layout_id = "combat_3",
                .next_room_ids = {6},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{MonsterKind::Melee, gameplay_config::room::CombatTierThreeMeleeMonsterCount},
                        RoomMonsterGroup{MonsterKind::Ranged, gameplay_config::room::CombatTierThreeRangedMonsterCount},
                    },
                },
            },
            DungeonRoomNode{
                .room_id = 6,
                .kind = DungeonRoomKind::Combat,
                .layout_id = "combat_4",
                .next_room_ids = {7},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{MonsterKind::Melee, gameplay_config::room::CombatTierFourMeleeMonsterCount},
                        RoomMonsterGroup{MonsterKind::Ranged, gameplay_config::room::CombatTierFourRangedMonsterCount},
                    },
                },
            },
            DungeonRoomNode{
                .room_id = 7,
                .kind = DungeonRoomKind::Combat,
                .layout_id = "combat_5",
                .next_room_ids = {8},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{MonsterKind::Melee, gameplay_config::room::CombatTierFiveMeleeMonsterCount},
                        RoomMonsterGroup{MonsterKind::Ranged, gameplay_config::room::CombatTierFiveRangedMonsterCount},
                    },
                },
            },
            DungeonRoomNode{
                .room_id = 8,
                .kind = DungeonRoomKind::Reward,
                .layout_id = "reward_large",
                .next_room_ids = {9},
            },
            DungeonRoomNode{
                .room_id = 9,
                .kind = DungeonRoomKind::Boss,
                .layout_id = "boss_large",
                .next_room_ids = {},
                .encounter = RoomEncounter{
                    .monster_groups = {
                        RoomMonsterGroup{
                            .kind = MonsterKind::Boss,
                            .count = gameplay_config::room::BossMonsterCount,
                        },
                    },
                },
            },
        }
    };
}
