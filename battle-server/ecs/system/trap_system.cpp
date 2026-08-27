#include "trap_system.hpp"

#include <algorithm>

#include "ecs/world.hpp"

void battle::ecs::trap_system(World& world, DeltaTime) {
    for (const auto entity : world.registry().pool<Trap>().entities()) {
        auto* trap = world.registry().try_get<Trap>(entity);
        const auto* transform = world.registry().try_get<Transform>(entity);
        const auto* collider = world.registry().try_get<Collider>(entity);
        if (!trap || !transform || !collider) {
            continue;
        }
        std::vector<Entity> current_targets{};
        for (const auto candidate : world.spatial_index().query_circle(transform->position, collider->radius)) {
            const auto* candidate_transform = world.registry().try_get<Transform>(candidate);
            const auto* candidate_collider = world.registry().try_get<Collider>(candidate);
            const auto* candidate_health = world.registry().try_get<Health>(candidate);
            if (!candidate_transform || !candidate_collider || !candidate_health) {
                continue;
            }
            if (candidate_collider->category != CollisionCategory::Monster && candidate_collider->category !=
                CollisionCategory::Player) {
                continue;
            }
            if (candidate_health->current_health <= 0) {
                continue;
            }
            const float dx = transform->position.x - candidate_transform->position.x;
            const float dy = transform->position.y - candidate_transform->position.y;
            const float radius_sum = collider->radius + candidate_collider->radius;
            if (dx * dx + dy * dy >= radius_sum * radius_sum) {
                continue;
            }
            current_targets.emplace_back(candidate);
            switch (trap->kind) {
            case TrapKind::Spikes: {
                if (std::ranges::find(trap->active_targets, candidate) == trap->active_targets.end()) {
                    world.add_damage_event(DamageEvent{
                        .source = entity,
                        .target = candidate,
                        .base_damage = gameplay_config::trap::spikes::Damage,
                        .modified_damage = gameplay_config::trap::spikes::Damage,
                        .source_kind = DamageSourceKind::Trap,
                    });
                }
                break;
            }

            case TrapKind::PoisonPool: {
                auto* candidate_status = world.registry().try_get<StatusEffects>(candidate);
                if (!candidate_status) {
                    continue;
                }
                if (candidate_status->poison.has_value()) {
                    auto& poison = candidate_status->poison.value();
                    // 持续站在毒池中只刷新状态有效期，保留 tick 进度以避免延后下一次伤害。
                    poison.source = entity;
                    poison.remaining_seconds = gameplay_config::trap::poison_pool::Duration;
                    poison.damage_per_tick = std::max(poison.damage_per_tick,
                                                      gameplay_config::trap::poison_pool::DamagePerTick);
                    poison.tick_interval_seconds = std::min(poison.tick_interval_seconds,
                                                            gameplay_config::trap::poison_pool::TickInterval);
                } else {
                    candidate_status->poison = std::make_optional(PoisonStatus{
                        .source = entity,
                        .remaining_seconds = gameplay_config::trap::poison_pool::Duration,
                        .tick_interval_seconds = gameplay_config::trap::poison_pool::TickInterval,
                        .tick_timer_seconds = DeltaTime{0},
                        .damage_per_tick = gameplay_config::trap::poison_pool::DamagePerTick,
                    });
                }
                break;
            }
            case TrapKind::Swamp: {
                auto* candidate_status = world.registry().try_get<StatusEffects>(candidate);
                if (!candidate_status) {
                    continue;
                }
                if (candidate_status->swamp.has_value()) {
                    candidate_status->swamp->remaining_seconds = gameplay_config::trap::swamp::Duration;
                    candidate_status->swamp->movement_multiplier = gameplay_config::trap::swamp::MovementMultiplier;
                } else {
                    candidate_status->swamp = std::make_optional(SwampStatus{
                        .remaining_seconds = gameplay_config::trap::swamp::Duration,
                        .movement_multiplier = gameplay_config::trap::swamp::MovementMultiplier,
                    });
                }
                break;
            }
            }
        }
        // 尖刺使用上一 tick 的目标集合判断重新进入；毒池和沼泽也复用该范围状态。
        trap->active_targets = std::move(current_targets);
    }
}
