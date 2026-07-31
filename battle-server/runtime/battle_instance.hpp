#pragma once

#include <string>
#include <cstdint>
#include <random>
#include <unordered_map>
#include <optional>

#include "battle_instance_types.hpp"
#include "player_input.hpp"
#include "ecs/world.hpp"
#include "gameplay/spawn_planner.hpp"
#include "gameplay/wave_planner.hpp"
#include "gameplay/monster_kind.hpp"

namespace battle {
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

        [[nodiscard]] const std::unordered_map<std::int64_t, PlayerBattleStats>& player_battle_stats() const {
            return player_battle_stats_;
        }

        [[nodiscard]] BattleSettlement settlement() const;

        [[nodiscard]] BattlePhase phase() const {
            return phase_;
        }

        [[nodiscard]] ecs::DeltaTime reward_selection_remaining() const {
            return reward_selection_.remaining_seconds;
        }

        [[nodiscard]] std::optional<ecs::PlayerProgress> player_progress(std::int64_t player_id) const;

        [[nodiscard]] std::optional<PlayerBlessingState> player_blessing_state(std::int64_t player_id) const;

        bool choose_blessing(std::int64_t player_id, int option_id);

    private:
        void spawn_next_wave_();

        void end_battle_(BattleEndReason reason);

        void consume_kill_events_();

        void tick_fighting_(ecs::DeltaTime delta_time);

        void tick_reward_selection_(ecs::DeltaTime delta_time);

        void start_reward_selection_();

        void apply_default_upgrade_choices_();

        void start_next_wave_or_end_();

        void grant_experience_(std::int64_t player_id, int experience);

        [[nodiscard]] int experience_for_monster_kind_(MonsterKind kind) const;

        [[nodiscard]] int experience_to_next_level_(int level) const;

        std::vector<BlessingOption> generate_blessing_options_(std::int64_t player_id);

        void add_or_level_up_blessing_(ecs::Entity player_entity, PlayerBlessingState& blessing_state,
                                       BlessingID blessing_id);



    private:
        std::string room_name_;
        ecs::World world_;
        SpawnPlanner spawn_planner_;
        std::unordered_map<std::int64_t, ecs::Entity> player_entities_;
        std::unordered_map<ecs::Entity, std::int64_t> entity_players_;
        std::unordered_map<std::int64_t, PlayerBattleStats> player_battle_stats_;
        BattleState state_;
        BattleEndReason end_reason_;
        std::size_t current_wave_;
        WaveConfig wave_config_;
        WavePlanner wave_planner_;
        BattlePhase phase_;
        RewardSelectionState reward_selection_;
        ProgressionConfig progression_config_;
        std::unordered_map<std::int64_t, PlayerBlessingState> player_blessings_;
        std::mt19937 reward_random_engine_;
    };
}
