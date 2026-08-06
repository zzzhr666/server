#pragma once
#include <cstddef>
#include <vector>
#include "monster_planner.hpp"
#include "spawn_planner.hpp"

namespace battle {
    /// @brief 默认战斗包含的波次数量。
    constexpr std::size_t WaveCount = 10;
    /// @brief 描述同一波中一种怪物的种类与数量。
    struct WaveMonsterGroup {
        MonsterKind kind;
        std::size_t count;
    };

    /// @brief 描述一波怪物的构成及基础属性倍率。
    struct WaveDefinition {
        std::vector<WaveMonsterGroup> groups;
        float health_multiplier = 1.0f;
        float move_speed_multiplier = 1.0f;
    };

    /// @brief 按顺序定义整局战斗的全部波次。
    struct WaveConfig {
        std::vector<WaveDefinition> waves;
    };

    /// @brief 返回本地默认波次配置。
    WaveConfig default_wave_config();

    /// @brief WavePlanner 将波次定义展开为带出生位置的怪物创建配置。
    class WavePlanner {
    public:
        /// @brief 按怪物组、属性倍率和出生规则规划一整波怪物。
        [[nodiscard]] std::vector<ecs::CreateMonsterConfig> plan_wave(const WaveDefinition& wave) const;

    private:
        SpawnPlanner spawn_planner_;
    };
}
