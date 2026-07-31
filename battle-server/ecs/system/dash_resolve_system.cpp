#include "dash_resolve_system.hpp"

#include "ecs/world.hpp"


void battle::ecs::dash_resolve_system(World& world, DeltaTime delta_time) {
    for (auto entity : world.registry().pool<DashRequest>().entities()) {
        auto request = world.registry().try_get<DashRequest>(entity);
        auto dash_cooldown = world.registry().try_get<DashCooldown>(entity);
        auto dashes = world.registry().try_get<Dash>(entity);
        auto intent = world.registry().try_get<DashIntent>(entity);
        if (!request || !dash_cooldown || !dashes || !intent) {
            continue;
        }
        intent->active = false;
        intent->dash_speed_multiplier = 1.0f;

        dash_cooldown->remaining_seconds -= delta_time;
        if (dash_cooldown->remaining_seconds < DeltaTime{0}) {
            dash_cooldown->remaining_seconds = DeltaTime{0};
        }

        if (!request->requested) {
            continue;
        }
        request->requested = false;
        if (dash_cooldown->remaining_seconds > DeltaTime{0}) {
            continue;
        }
        intent->active = true;
        intent->dash_speed_multiplier = dashes->dash_speed_multiplier;
        dash_cooldown->remaining_seconds = dashes->cooldown_seconds;
    }
}
