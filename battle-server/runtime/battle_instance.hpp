#pragma once

#include <string>
#include <vector>
#include <cstdint>
#include <unordered_map>

#include "ecs/world.hpp"
#include "gameplay/spawn_planner.hpp"


namespace battle {
    struct BattleInstanceConfig {
        std::string room_name;
        std::vector<std::int64_t> player_ids;
    };

    class BattleInstance {
    public:
        explicit BattleInstance(BattleInstanceConfig config);

        void tick(ecs::DeltaTime delta_time);

        bool receive_input(std::int64_t player_id, float x, float y);

        [[nodiscard]] ecs::WorldSnapshot snapshot() const;

    private:
        std::string room_name_;
        ecs::World world_;
        SpawnPlanner spawn_planner_;
        std::unordered_map<std::int64_t, ecs::Entity> player_entities_;
    };
}
