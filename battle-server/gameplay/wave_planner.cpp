#include "wave_planner.hpp"

#include <array>
#include <ranges>

namespace {
    struct DefaultWaveComposition {
        std::size_t melee_count;
        std::size_t ranged_count;
    };

    constexpr std::array<DefaultWaveComposition, battle::WaveCount> DefaultWaveCompositions{
        DefaultWaveComposition{.melee_count = 7, .ranged_count = 0},
        DefaultWaveComposition{.melee_count = 7, .ranged_count = 1},
        DefaultWaveComposition{.melee_count = 8, .ranged_count = 2},
        DefaultWaveComposition{.melee_count = 8, .ranged_count = 3},
        DefaultWaveComposition{.melee_count = 9, .ranged_count = 4},
        DefaultWaveComposition{.melee_count = 10, .ranged_count = 4},
        DefaultWaveComposition{.melee_count = 11, .ranged_count = 5},
        DefaultWaveComposition{.melee_count = 11, .ranged_count = 6},
        DefaultWaveComposition{.melee_count = 12, .ranged_count = 7},
        DefaultWaveComposition{.melee_count = 12, .ranged_count = 8},
    };
}

battle::WaveConfig battle::default_wave_config() {
    WaveConfig config;
    for (std::size_t i = 0; i < WaveCount; ++i) {
        const auto& composition = DefaultWaveCompositions[i];
        WaveDefinition wave_definition{
            .groups = {
                WaveMonsterGroup{
                    .kind = MonsterKind::Melee,
                    .count = composition.melee_count,
                },
            },
            .health_multiplier = 1.0f + 0.2f * static_cast<float>(i),
            .move_speed_multiplier = 1.0f,
        };
        if (composition.ranged_count > 0) {
            wave_definition.groups.emplace_back(WaveMonsterGroup{
                .kind = MonsterKind::Ranged,
                .count = composition.ranged_count,
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
