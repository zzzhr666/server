#include "wave_planner.hpp"

#include <ranges>

battle::WaveConfig battle::default_wave_config() {
    WaveConfig config;
    for (std::size_t i = 0; i < WaveCount; ++i) {
        config.waves.emplace_back(WaveDefinition{
            .groups = {
                WaveMonsterGroup{
                    .kind = MonsterKind::Melee,
                    .count = 3 + i
                }
            },
            .health_multiplier = 1.0f + 0.2f * static_cast<float>(i),
            .move_speed_multiplier = 1.0f,
        });
    }
    return config;
}

std::vector<battle::ecs::CreateMonsterConfig> battle::WavePlanner::plan_wave(const WaveDefinition& wave) const {
    std::vector<battle::ecs::CreateMonsterConfig> result;
    std::size_t total_count = 0;
    for (const auto& [_, count] : wave.groups) {
        total_count += count;
    }
    std::size_t monster_index = 0;
    for (const auto& [kind, count] : wave.groups) {
        auto definition = monster_definition(kind);
        for (std::size_t i = 0; i < count; ++i, ++monster_index) {
            auto config = spawn_planner_.monster_spawn(monster_index, total_count);
            config.max_health = static_cast<int>(static_cast<float>(definition.base_health) * wave.health_multiplier);
            config.move_speed = definition.base_move_speed * wave.move_speed_multiplier;
            result.emplace_back(config);
        }
    }
    return result;
}
