#include "wave_planner.hpp"

#include <ranges>

battle::WaveConfig battle::default_wave_config() {
    WaveConfig config;
    for (std::size_t i = 0; i < WaveCount; ++i) {
        const std::size_t total_count = 3 + i;
        const std::size_t ranged_count = i < 2 ? 0 : 1 + (i - 2) / 3;
        const std::size_t melee_count = total_count - ranged_count;
        WaveDefinition wave_definition{
            .groups = {
                WaveMonsterGroup{
                    .kind = MonsterKind::Melee,
                    .count = melee_count,
                },
            },
            .health_multiplier = 1.0f + 0.2f * static_cast<float>(i),
            .move_speed_multiplier = 1.0f,
        };
        if (ranged_count > 0) {
            wave_definition.groups.emplace_back(WaveMonsterGroup{
                .kind = MonsterKind::Ranged,
                .count = ranged_count,
            });
        }
        config.waves.emplace_back(wave_definition);
    }
    return config;
}

std::vector<battle::ecs::CreateMonsterConfig> battle::WavePlanner::plan_wave(const WaveDefinition& wave) const {
    std::vector<ecs::CreateMonsterConfig> result;
    std::size_t total_count = 0;
    for (const auto& [_, count] : wave.groups) {
        total_count += count;
    }
    std::size_t monster_index = 0;
    for (const auto& [kind, count] : wave.groups) {
        auto definition = monster_definition(kind);
        for (std::size_t i = 0; i < count; ++i, ++monster_index) {
            auto config = spawn_planner_.monster_spawn(monster_index, total_count);
            config.kind = definition.kind;
            config.max_health = static_cast<int>(static_cast<float>(definition.base_health) * wave.health_multiplier);
            config.move_speed = definition.base_move_speed * wave.move_speed_multiplier;
            config.attack = definition.base_attack;
            config.kiting_ai = definition.kiting_ai;
            result.emplace_back(config);
        }
    }
    return result;
}
