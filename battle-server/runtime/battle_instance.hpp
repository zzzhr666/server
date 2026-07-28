#pragma once

#include <string>
#include <vector>
#include <cstdint>
#include <unordered_map>
#include <optional>

#include "player_input.hpp"
#include "ecs/world.hpp"
#include "gameplay/spawn_planner.hpp"
#include "gameplay/wave_planner.hpp"


namespace battle {
    struct BattleInstanceConfig {
        std::string room_name;
        std::vector<std::int64_t> player_ids;
        WaveConfig wave_config = default_wave_config();
        std::optional<ecs::CreatePlayerConfig> player_config_override;
        ecs::WorldBounds world_bounds = ecs::WorldBounds{
            .min_x = -20.0f,
            .max_x = 20.0f,
            .min_y = -20.0f,
            .max_y = 20.0f,
        };
    };

    enum class BattleState : std::uint8_t {
        Running,
        Ended
    };

    enum class BattleEndReason:std::uint8_t {
        None,
        Victory,
        Defeat,
    };

    struct BattleEntitySnapshot {
        ecs::Entity entity;
        ecs::EntityKind kind;
        std::int64_t player_id;
        float x_position;
        float y_position;
        float x_direction;
        float y_direction;
        int current_health;
        int max_health;
    };

    struct BattleWorldSnapshot {
        std::vector<BattleEntitySnapshot> entities;
    };


    class BattleInstance {
    public:
        explicit BattleInstance(BattleInstanceConfig config);

        void tick(ecs::DeltaTime delta_time);

        bool receive_input(std::int64_t player_id, PlayerInput input);

        [[nodiscard]] BattleWorldSnapshot snapshot() const;

        [[nodiscard]] std::size_t current_wave() const {
            return current_wave_;
        }

        [[nodiscard]] BattleState state() const {
            return state_;
        }

        [[nodiscard]] BattleEndReason end_reason() const {
            return end_reason_;
        }

        [[nodiscard]] bool ended() const {
            return state_ == BattleState::Ended;
        }

    private:
        void spawn_next_wave_();

        void end_battle_(BattleEndReason reason);

    private:
        std::string room_name_;
        ecs::World world_;
        SpawnPlanner spawn_planner_;
        std::unordered_map<std::int64_t, ecs::Entity> player_entities_;
        std::unordered_map<ecs::Entity, std::int64_t> entity_players_;
        BattleState state_;
        BattleEndReason end_reason_;
        std::size_t current_wave_;
        WaveConfig wave_config_;
        WavePlanner wave_planner_;
    };
}
