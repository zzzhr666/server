#pragma once

#include <vector>

#include "ecs/world.hpp"


namespace battle {
    struct RoomEncounter;
    struct RoomLayout;

    class RoomEncounterPlanner {
    public:
        /// @brief 将房间遭遇与布局出生点转换为怪物创建配置。
        static std::vector<ecs::CreateMonsterConfig> plan_encounter(
            const RoomEncounter& encounter, const RoomLayout& layout);
    };
}
