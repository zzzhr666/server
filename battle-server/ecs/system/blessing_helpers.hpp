#pragma once

#include "ecs/component/components.hpp"
#include "ecs/entity/entity.hpp"
#include "gameplay/blessing.hpp"

namespace battle::ecs {
    class World;

    /// @brief 查找实体指定祝福堆叠，不存在时返回 nullptr。
    [[nodiscard]] const BlessingStack* find_blessing(const World& world, Entity entity, BlessingID blessing_id);

    /// @brief 返回实体指定祝福等级，不存在时返回零。
    [[nodiscard]] int blessing_level(const World& world, Entity entity, BlessingID blessing_id);

    /// @brief 使用 World 的可复现随机源执行百分比判定。
    [[nodiscard]] bool roll_percent(World& world, int chance_percent);
}
