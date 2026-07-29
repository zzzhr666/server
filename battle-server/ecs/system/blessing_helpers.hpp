#pragma once

#include "ecs/component/components.hpp"
#include "ecs/entity/entity.hpp"
#include "gameplay/blessing.hpp"

namespace battle::ecs {
    class World;

    [[nodiscard]] const BlessingStack* find_blessing(const World& world, Entity entity, BlessingID blessing_id);

    [[nodiscard]] int blessing_level(const World& world, Entity entity, BlessingID blessing_id);

    [[nodiscard]] bool roll_percent(World& world, int chance_percent);
}
