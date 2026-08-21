#include "status_effect_system.hpp"

#include "ecs/world.hpp"

namespace {
    void resolve_burn_on_hit(battle::ecs::World& world, battle::ecs::Entity entity,
                             std::vector<battle::ecs::BurnStatus>& status_effect,
                             battle::ecs::DeltaTime delta_seconds) {
        for (auto& status : status_effect) {
            status.remaining_seconds -= delta_seconds;
            status.tick_timer_seconds += delta_seconds;
            while (status.tick_timer_seconds >= status.tick_interval_seconds) {
                world.add_damage_event(battle::ecs::DamageEvent{
                    .source = status.source,
                    .target = entity,
                    .base_damage = status.damage_per_tick,
                    .modified_damage = status.damage_per_tick,
                    .source_kind = battle::ecs::DamageSourceKind::Burn,
                });
                status.tick_timer_seconds -= status.tick_interval_seconds;
            }
        }
        std::erase_if(status_effect, [](const battle::ecs::BurnStatus& status) {
            return status.remaining_seconds <= battle::ecs::DeltaTime{0};
        });
    }

    void resolve_freeze_on_hit(battle::ecs::World& world, battle::ecs::Entity entity,
                               std::optional<battle::ecs::FreezeStatus>& freeze_status,
                               battle::ecs::DeltaTime delta_seconds) {
        freeze_status.value().remaining_seconds -= delta_seconds;
        if (freeze_status.value().remaining_seconds <= battle::ecs::DeltaTime{0}) {
            freeze_status.reset();
        }
    }

    void resolve_poison(battle::ecs::World& world, battle::ecs::Entity entity,
                        std::optional<battle::ecs::PoisonStatus>& poison_status,
                        battle::ecs::DeltaTime delta_seconds) {
        auto& status = poison_status.value();
        status.remaining_seconds -= delta_seconds;
        status.tick_timer_seconds += delta_seconds;
        while (status.tick_timer_seconds >= status.tick_interval_seconds) {
            world.add_damage_event(battle::ecs::DamageEvent{
                .source = status.source,
                .target = entity,
                .base_damage = status.damage_per_tick,
                .modified_damage = status.damage_per_tick,
                .source_kind = battle::ecs::DamageSourceKind::Trap,
            });
            status.tick_timer_seconds -= status.tick_interval_seconds;
        }
        if (status.remaining_seconds <= battle::ecs::DeltaTime{0.0f}) {
            poison_status.reset();
        }
    }

    void resolve_swamp(battle::ecs::World& world, battle::ecs::Entity entity,
                       std::optional<battle::ecs::SwampStatus>& swamp_status,
                       battle::ecs::DeltaTime delta_time) {
        swamp_status.value().remaining_seconds -= delta_time;
        if (swamp_status.value().remaining_seconds <= battle::ecs::DeltaTime{0.0f}) {
            swamp_status.reset();
        }
    }
}


void battle::ecs::status_effect_system(World& world, DeltaTime delta_time) {
    for (auto entity : world.registry().pool<StatusEffects>().entities()) {
        auto status_effect = world.registry().try_get<StatusEffects>(entity);
        if (!status_effect) {
            continue;
        }

        if (!status_effect->burns.empty()) {
            resolve_burn_on_hit(world, entity, status_effect->burns, delta_time);
        }

        if (status_effect->freeze.has_value()) {
            resolve_freeze_on_hit(world, entity, status_effect->freeze, delta_time);
        }

        if (status_effect->poison.has_value()) {
            resolve_poison(world, entity, status_effect->poison, delta_time);
        }
        if (status_effect->swamp.has_value()) {
            resolve_swamp(world, entity, status_effect->swamp, delta_time);
        }
    }
}
