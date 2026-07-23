#include "battle_instance.hpp"


battle::BattleInstance::BattleInstance(BattleInstanceConfig config)
    : room_name_(std::move(config.room_name)) {
    for (std::size_t i = 0; i < config.player_ids.size(); ++i) {
        auto spawn_config = spawn_planner_.player_spawn(config.player_ids[i], i);
        world_.create_player(spawn_config);
    }
}

void battle::BattleInstance::tick(float delta_seconds) {
    world_.tick(delta_seconds);
}

bool battle::BattleInstance::receive_input(std::int64_t player_id, float x, float y) {
    return world_.set_move_input(player_id, x, y);
}

battle::ecs::WorldSnapshot battle::BattleInstance::snapshot() const {
    return world_.snapshot();
}
