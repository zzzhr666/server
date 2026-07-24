#include "battle_instance.hpp"


battle::BattleInstance::BattleInstance(BattleInstanceConfig config)
    : room_name_(std::move(config.room_name)) {
    for (std::size_t i = 0; i < config.player_ids.size(); ++i) {
        if (player_entities_.contains(config.player_ids[i])) {
            continue;
        }
        auto spawn_config = spawn_planner_.player_spawn(i);
        auto entity = world_.create_player(spawn_config);
        player_entities_.emplace(config.player_ids[i], entity);
    }
}

void battle::BattleInstance::tick(ecs::DeltaTime delta_time) {
    world_.tick(delta_time);
}

bool battle::BattleInstance::receive_input(std::int64_t player_id, float x, float y) {
    auto it = player_entities_.find(player_id);
    if (it == player_entities_.end()) {
        return false;
    }
    return world_.set_move_input(it->second, x, y);
}

battle::ecs::WorldSnapshot battle::BattleInstance::snapshot() const {
    return world_.snapshot();
}
