#pragma once

#include <cstddef>
#include <vector>

#include "monster_kind.hpp"

namespace battle {
    /// @brief 描述一次房间遭遇中的一组同类怪物。
    struct RoomMonsterGroup {
        MonsterKind kind{MonsterKind::Melee};
        std::size_t count{};
    };

    /// @brief 描述一个房间只生成一次的怪物遭遇。
    struct RoomEncounter {
        std::vector<RoomMonsterGroup> monster_groups;
    };

}
