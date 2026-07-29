#include "blessing_trigger_system.hpp"

#include <algorithm>

#include "blessing_config.hpp"
#include "blessing_helpers.hpp"
#include "ecs/world.hpp"
namespace {
    void handle_life_steal(battle::ecs::World& world, const battle::ecs::DamageAppliedEvent& event) {
        auto health = world.health().try_get(event.source);
        if (!health) {
            return;
        }
        auto blessing = battle::ecs::find_blessing(world, event.source, battle::BlessingID::LifeSteal);
        if (!blessing) {
            return;
        }
        const int heal_amount = event.amount * battle::ecs::life_steal_percent(blessing->level) / 100;
        health->current_health = std::clamp(health->current_health + heal_amount, 0, health->max_health);
    }

    void handle_burn_on_hit(battle::ecs::World& world, const battle::ecs::DamageAppliedEvent& event) {
        auto blessing = find_blessing(world, event.source, battle::BlessingID::BurnOnHit);
        if (!blessing) {
            return;
        }
        auto status_effect = world.status_effects().try_get(event.target);
        if (!status_effect) {
            return;
        }

        status_effect->burns.emplace_back(battle::ecs::BurnStatus{
            .source = event.source,
            .remaining_seconds = battle::ecs::burn_duration_seconds(blessing->level),
            .tick_interval_seconds = battle::ecs::BurnOnHitConfig::TickIntervalSeconds,
            .tick_timer_seconds = battle::ecs::DeltaTime{0.0f},
            .damage_per_tick = battle::ecs::burn_damage_per_tick(blessing->level),
        });
    }

    void handle_freeze_on_hit(battle::ecs::World& world, const battle::ecs::DamageAppliedEvent& event) {
        auto blessing = battle::ecs::find_blessing(world, event.source, battle::BlessingID::FreezeOnHit);
        if (!blessing) {
            return;
        }
        auto status_effect = world.status_effects().try_get(event.target);
        if (!status_effect) {
            return;
        }
        if (!battle::ecs::roll_percent(world, battle::ecs::freeze_percent(blessing->level))) {
            return;
        }
        const auto duration = battle::ecs::freeze_duration_seconds(blessing->level);
        if (!status_effect->freeze || status_effect->freeze->remaining_seconds < duration) {
            status_effect->freeze = battle::ecs::FreezeStatus{
                .remaining_seconds = duration,
            };
        }
    }
}
namespace battle::ecs {


void blessing_trigger_system(World& world, DeltaTime) {
    for (const auto& event : world.damage_applied_events()) {
        if (event.source_kind != DamageSourceKind::Attack || event.amount <= 0) {
            continue;
        }
        handle_life_steal(world, event);
        handle_burn_on_hit(world, event);
        handle_freeze_on_hit(world, event);
    }

    world.clear_damage_applied_events();
}
}
