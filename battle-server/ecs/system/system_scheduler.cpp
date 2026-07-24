#include "ecs/system/system_scheduler.hpp"

battle::ecs::SystemScheduler::SystemScheduler() = default;

battle::ecs::SystemScheduler::SystemScheduler(std::initializer_list<sysFunc> funcs) {
    for (auto& func : funcs) {
        add_system(func);
    }
}

void battle::ecs::SystemScheduler::add_system(sysFunc func) {
    systems_.emplace_back(std::move(func));
}

void battle::ecs::SystemScheduler::tick(World& world, DeltaTime delta_time) {
    for (auto& system : systems_) {
        system(world, delta_time);
    }
}
