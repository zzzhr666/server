#pragma once
#include <functional>
#include <vector>

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    using sysFunc = std::function<void(ecs::World&, DeltaTime)>;
    class SystemScheduler {
    public:
        SystemScheduler();
        SystemScheduler(std::initializer_list<sysFunc> funcs);
        void add_system(sysFunc func);
        void tick(World& world, DeltaTime delta_time);
    private:
        std::vector<sysFunc> systems_;
    };
}
