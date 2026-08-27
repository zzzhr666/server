#include "damage_modify_system.hpp"

#include "blessing_config.hpp"
#include "ecs/system/blessing_helpers.hpp"
#include "ecs/world.hpp"


namespace {
    void handle_critical_strike(battle::ecs::World& world, battle::ecs::DamageEvent& damage_event) {
        const auto* blessing = battle::ecs::find_blessing(world, damage_event.source,
                                                          battle::BlessingID::CriticalStrike);
        if (!blessing) {
            return;
        }
        if (!battle::ecs::roll_percent(world, battle::ecs::critical_strike_percent(blessing->level))) {
            return;
        }
        damage_event.modified_damage = damage_event.modified_damage *
            battle::ecs::critical_strike_damage_percent(blessing->level) / 100;
    }

    void handle_heavy_strike(battle::ecs::World& world, battle::ecs::DamageEvent& damage_event) {
        const auto* blessing = battle::ecs::find_blessing(world, damage_event.source, battle::BlessingID::HeavyStrike);
        if (!blessing) {
            return;
        }
        auto health = world.registry().try_get<battle::ecs::Health>(damage_event.target);
        if (!health || health->current_health < health->max_health) {
            return;
        }
        damage_event.modified_damage +=
            damage_event.base_damage * battle::ecs::heavy_strike_extra_damage_percent(blessing->level) / 100;
    }

    void handle_revenge(battle::ecs::World& world, battle::ecs::DamageEvent& damage_event) {
        const auto* blessing = battle::ecs::find_blessing(world, damage_event.source, battle::BlessingID::Revenge);
        if (!blessing) {
            return;
        }
        auto health = world.registry().try_get<battle::ecs::Health>(damage_event.source);
        if (!health || health->current_health * 2 >= health->max_health) {
            return;
        }
        damage_event.modified_damage +=
            damage_event.base_damage * battle::ecs::revenge_extra_damage_percent(blessing->level) / 100;
    }
}

void battle::ecs::damage_modify_system(World& world, DeltaTime) {
    for (auto& damage_event : world.damage_events()) {
        if (damage_event.source_kind != DamageSourceKind::Attack) {
            continue;
        }
        handle_critical_strike(world, damage_event);
        handle_heavy_strike(world, damage_event);
        handle_revenge(world, damage_event);
    }
}
