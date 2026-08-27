#include "damage_system.hpp"

#include <algorithm>

#include "ecs/world.hpp"

namespace {
    int get_final_damage(const battle::ecs::CharacterStats* stats,int base_damage) {
        if (base_damage < 0) {
            return 0;
        }
        const int armor = std::max(stats->armor, -99);
        const auto scaled_damage = static_cast<std::int64_t>(base_damage) * 100;
        return static_cast<int>(scaled_damage / (100 + armor));
    }
}

void battle::ecs::damage_system(World& world, DeltaTime) {
    for (const auto& event : world.damage_events()) {
        auto* health = world.registry().try_get<Health>(event.target);
        const auto* character_stat = world.registry().try_get<CharacterStats>(event.target);
        if (!health || !character_stat) {
            continue;
        }
        const int final_damage = get_final_damage(character_stat, event.modified_damage);
        const int before_health = health->current_health;
        health->current_health = std::clamp(health->current_health - final_damage, 0, health->max_health);
        if (final_damage > 0) {
            world.add_damage_applied_event({
                .source = event.source,
                .target = event.target,
                .amount = final_damage,
                .source_kind = event.source_kind,
                .context = event.context,
            });
        }
        if (before_health > 0 && health->current_health == 0) {
            auto transform = world.registry().try_get<Transform>(event.target);
            if (!transform) {
                continue;
            }
            if (world.registry().has<PlayerController>(event.target)) {
                world.add_death_event(DeathEvent{
                    .victim = event.target,
                    .killer = event.source,
                    .kind = DeathEntityKind::Player,
                    .position = transform->position,
                    .direction = transform->direction,
                    .monster_kind = std::nullopt,
                });
            }
            if (world.registry().has<MonsterController>(event.target)) {
                auto identity = world.registry().try_get<MonsterIdentity>(event.target);
                if (!identity) {
                    continue;
                }
                world.add_death_event(DeathEvent{
                    .victim = event.target,
                    .killer = event.source,
                    .kind = DeathEntityKind::Monster,
                    .position = transform->position,
                    .direction = transform->direction,
                    .monster_kind = identity->kind,
                });
                world.add_kill_event({
                    .killer = event.source,
                    .victim = event.target,
                    .monster_kind = identity->kind,
                });
            }
        }
    }
    world.clear_damage_events();
}
