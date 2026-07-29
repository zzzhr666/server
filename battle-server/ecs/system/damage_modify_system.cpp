#include "damage_modify_system.hpp"

#include "blessing_config.hpp"
#include "ecs/system/blessing_helpers.hpp"
#include "ecs/world.hpp"

void battle::ecs::damage_modify_system(World& world, DeltaTime) {
    for (auto& damage_event : world.damage_events()) {
        if (damage_event.source_kind != DamageSourceKind::Attack) {
            continue;
        }
        const auto* blessing = find_blessing(world, damage_event.source, BlessingID::CriticalStrike);
        if (!blessing) {
            continue;
        }
        if (!roll_percent(world, critical_strike_percent(blessing->level))) {
            continue;
        }
        damage_event.modified_damage *= CriticalStrikeConfig::DamageMultiplier;
    }
}
