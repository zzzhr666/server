#pragma once
#include <cstddef>
#include <vector>
#include "monster_planner.hpp"
#include "spawn_planner.hpp"

namespace battle {
    constexpr std::size_t WaveCount = 10;
    struct WaveMonsterGroup {
        MonsterKind kind;
        std::size_t count;
    };

    struct WaveDefinition {
        std::vector<WaveMonsterGroup> groups;
        float health_multiplier = 1.0f;
        float move_speed_multiplier = 1.0f;
    };

    struct WaveConfig {
        std::vector<WaveDefinition> waves;
    };

    WaveConfig default_wave_config();

    class WavePlanner {
    public:
        [[nodiscard]] std::vector<ecs::CreateMonsterConfig> plan_wave(const WaveDefinition& wave) const;

    private:
        SpawnPlanner spawn_planner_;
    };
}
