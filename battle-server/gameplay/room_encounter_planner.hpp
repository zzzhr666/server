#pragma once

#include <vector>

#include "ecs/world.hpp"


namespace battle {
    struct RoomEncounter;
    struct RoomLayout;

    class RoomEncounterPlanner {
    public:
        static std::vector<ecs::CreateMonsterConfig> plan_encounter(
            const RoomEncounter& encounter, const RoomLayout& layout);
    };
}
