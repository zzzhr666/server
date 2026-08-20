#include "pre_damage_blessing_system.hpp"

#include <vector>
#include <algorithm>

#include "blessing_config.hpp"
#include "blessing_helpers.hpp"
#include "combat_targeting.hpp"
#include "ecs/world.hpp"

namespace {
    std::vector<battle::ecs::Entity> find_chain_lightning_targets(const battle::ecs::World& world,
                                                                  battle::ecs::Entity owner,
                                                                  battle::ecs::Entity primary_target,
                                                                  int max_target_count, float jump_radius) {
        if (max_target_count <= 0 || jump_radius <= 0.0f) {
            return {};
        }
        std::vector<battle::ecs::Entity> targets;
        targets.reserve(static_cast<std::size_t>(max_target_count));
        battle::ecs::Entity anchor = primary_target;
        const float max_distance_squared = jump_radius * jump_radius;
        for (int jump_index = 0; jump_index < max_target_count; ++jump_index) {
            const auto anchor_transform = world.registry().try_get<battle::ecs::Transform>(anchor);
            if (!anchor_transform) {
                break;
            }
            battle::ecs::Entity next_target{};
            float best_distance_squared = 0.0f;

            for (const auto candidate : world.registry().pool<battle::ecs::Health>().entities()) {
                if (candidate == owner || candidate == primary_target) {
                    continue;
                }
                if (std::ranges::find(targets, candidate) != targets.end()) {
                    continue;
                }
                if (!battle::ecs::is_enemy(world, owner, candidate)) {
                    continue;
                }
                const auto health = world.registry().try_get<battle::ecs::Health>(candidate);
                const auto transform = world.registry().try_get<battle::ecs::Transform>(candidate);
                if (!transform || !health || health->current_health <= 0) {
                    continue;
                }
                const float delta_x = transform->position.x - anchor_transform->position.x;
                const float delta_y = transform->position.y - anchor_transform->position.y;
                const float distance_squared = delta_x * delta_x + delta_y * delta_y;
                if (distance_squared > max_distance_squared) {
                    continue;
                }
                if (!next_target || distance_squared < best_distance_squared || (distance_squared ==
                    best_distance_squared && candidate < next_target)) {
                    next_target = candidate;
                    best_distance_squared = distance_squared;
                }
            }
            if (!next_target) {
                break;
            }
            targets.emplace_back(next_target);
            anchor = next_target;
        }

        return targets;
    }
}

void battle::ecs::pre_damage_blessing_system(World& world, DeltaTime delta_seconds) {
    std::vector<DamageEvent> chain_events;
    for (const auto& event : world.damage_events()) {
        if (event.source_kind != DamageSourceKind::Attack ||
            event.modified_damage <= 0 ||
            !event.context.action_state) {
            continue;
        }
        auto blessing = find_blessing(world,event.context.owner,BlessingID::ChainLightning);
        if (!blessing) {
            continue;
        }
        if (event.context.action_state->has_triggered(CombatProc::ChainLightning)) {
            continue;
        }
        const auto targets = find_chain_lightning_targets(world,event.context.owner,event.target,chain_lightning_target_count(blessing->level),ChainLightningConfig::JumpRadius);
        if (targets.empty()) {
            continue;
        }
        int chain_damage = event.modified_damage * chain_lightning_damage_percent(blessing->level) / 100;
        if (chain_damage < 0) {
            continue;
        }
        if (!event.context.action_state->try_trigger(CombatProc::ChainLightning)) {
            continue;
        }
        auto chain_effect_id = world.create_combat_effect();
        Entity anchor = event.target;
        for (auto target : targets) {
            auto context = event.context;
            context.emitter = anchor;
            context.effect_id = chain_effect_id;
            chain_events.emplace_back(DamageEvent{
                .source = event.context.owner,
                .target = target,
                .base_damage = chain_damage,
                .modified_damage =  chain_damage,
                .source_kind = DamageSourceKind::ChainLightning,
                .context = std::move(context),
            });
            anchor = target;
        }
    }
    for (auto& event : chain_events) {
        world.add_damage_event(std::move(event));
    }
}
