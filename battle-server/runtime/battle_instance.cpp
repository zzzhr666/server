#include "battle_instance.hpp"


battle::BattleInstance::BattleInstance(BattleInstanceConfig config)
    : room_name_(std::move(config.room_name)),
      state_(BattleState::Running),
      end_reason_(BattleEndReason::None),
      current_wave_(0),
      wave_config_(std::move(config.wave_config)),
      wave_planner_() {
    for (std::size_t i = 0; i < config.player_ids.size(); ++i) {
        if (player_entities_.contains(config.player_ids[i])) {
            continue;
        }
        auto spawn_config = spawn_planner_.player_spawn(i);

        if (config.player_config_override.has_value()) {
            auto override_config = config.player_config_override.value();
            override_config.position = spawn_config.position;
            spawn_config = override_config;
        }
        auto entity = world_.create_player(spawn_config);
        player_entities_.emplace(config.player_ids[i], entity);
    }
}

void battle::BattleInstance::tick(ecs::DeltaTime delta_time) {
    if (state_ == BattleState::Ended) {
        return;
    }
    if (current_wave_ == 0) {
        spawn_next_wave_();
    }
    world_.tick(delta_time);
    if (!world_.has_living_players()) {
        end_battle_(BattleEndReason::Defeat);
        return;
    }
    if (!world_.has_living_monsters()) {
        if (current_wave_ >= wave_config_.waves.size()) {
            end_battle_(BattleEndReason::Victory);
            return;
        }
        spawn_next_wave_();
    }
}

bool battle::BattleInstance::receive_input(std::int64_t player_id, PlayerInput input) {
    auto it = player_entities_.find(player_id);
    if (it == player_entities_.end()) {
        return false;
    }
    return world_.set_player_command(it->second, ecs::PlayerCommand{
                                         .move_x = input.move_x,
                                         .move_y = input.move_y,
                                         .attack_requested = input.attack_requested,
                                         .dash_requested = input.dash_requested,
                                     });
}

battle::ecs::WorldSnapshot battle::BattleInstance::snapshot() const {
    return world_.snapshot();
}

void battle::BattleInstance::spawn_next_wave_() {
    if (current_wave_ >= wave_config_.waves.size()) {
        return;
    }
    auto monster_configs = wave_planner_.plan_wave(wave_config_.waves[current_wave_]);
    for (const auto& monster_config : monster_configs) {
        world_.create_monster(monster_config);
    }
    ++current_wave_;
}

void battle::BattleInstance::end_battle_(BattleEndReason reason) {
    state_ = BattleState::Ended;
    end_reason_ = reason;
}
