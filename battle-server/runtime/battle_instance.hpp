#pragma once

#include <string>
#include <vector>
#include <cstdint>

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

        void tick(float delta_seconds);

        bool receive_input(std::int64_t player_id, float x, float y);

        ecs::WorldSnapshot snapshot() const;

    private:
        std::string room_name_;
        ecs::World world_;
        SpawnPlanner spawn_planner_;
    };
}
