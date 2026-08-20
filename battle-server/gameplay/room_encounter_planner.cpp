#include "room_encounter_planner.hpp"

#include "monster_planner.hpp"
#include "room_encounter.hpp"
#include "room_layout.hpp"

std::vector<battle::ecs::CreateMonsterConfig> battle::RoomEncounterPlanner::plan_encounter(
    const RoomEncounter& encounter, const RoomLayout& layout) {
    std::size_t monster_index{0};
    std::vector<battle::ecs::CreateMonsterConfig> monster_configs;
    for (const auto& [kind, count] : encounter.monster_groups) {
        auto definition = monster_definition(kind);
        for (std::size_t i = 0; i < count; ++i) {
            const auto& spawn_point = layout.monster_spawn_points[monster_index % layout.monster_spawn_points.size()];
            monster_configs.emplace_back(battle::ecs::CreateMonsterConfig{
                .kind = definition.kind,
                .position = spawn_point,
                .max_health = definition.base_health,
                .move_speed = definition.base_move_speed,
                .attack = definition.base_attack,
                .kiting_ai = definition.kiting_ai,
            });
            monster_index++;
        }
    }

    return monster_configs;
}
