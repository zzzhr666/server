#include "attack_resolve_system.hpp"

#include "ecs/world.hpp"


namespace {
    void tick_attack_cooldown(battle::ecs::AttackCooldown& cooldown, battle::ecs::DeltaTime delta_time) {
        cooldown.remaining_seconds -= delta_time;
        if (cooldown.remaining_seconds < battle::ecs::DeltaTime{0}) {
            cooldown.remaining_seconds = battle::ecs::DeltaTime{0};
        }
    }

    void finish_attack(battle::ecs::AttackState& state) {
        state.phase = battle::ecs::AttackPhase::Idle;
        state.phase_remaining = battle::ecs::DeltaTime{0.0f};
        state.hit_targets.clear();
        state.locked_direction = {};
        state.context = {};
        state.projectile_spawned = false;
    }

    void enter_recovery_or_idle(const battle::ecs::AttackDefinition& attack, battle::ecs::AttackState& state) {
        if (attack.recovery_seconds > battle::ecs::DeltaTime{0}) {
            state.phase = battle::ecs::AttackPhase::Recovery;
            state.phase_remaining = attack.recovery_seconds;
            return;
        }
        finish_attack(state);
    }

    void enter_active(const battle::ecs::AttackDefinition& attack, battle::ecs::AttackState& state) {
        if (attack.active_seconds > battle::ecs::DeltaTime{0}) {
            state.phase = battle::ecs::AttackPhase::Active;
            state.phase_remaining = attack.active_seconds;
            return;
        }
        enter_recovery_or_idle(attack, state);
    }


    void begin_attack(battle::ecs::World& world, battle::ecs::Entity entity,
                      const battle::ecs::AttackDefinition& attack, const battle::ecs::Transform& transform,
                      battle::ecs::AttackCooldown& cooldown, battle::ecs::AttackState& state) {
        const battle::ecs::CombatContext context{
            .owner = entity,
            .emitter = entity,
            .action_state = world.create_combat_action(),
            .effect_id = world.create_combat_effect(),
        };
        state.context = context;
        state.locked_direction = transform.direction;
        state.hit_targets.clear();
        state.projectile_spawned = false;
        cooldown.remaining_seconds = attack.cooldown_seconds;
        world.add_attack_event(battle::ecs::AttackEvent{
            .attacker = entity,
            .kind = attack.kind,
            .direction = transform.direction,
            .action_id = context.action_state->action_id,
            .windup_seconds = attack.windup_seconds,
            .active_seconds = attack.active_seconds,
            .recovery_seconds = attack.recovery_seconds,
        });
        if (attack.windup_seconds > battle::ecs::DeltaTime{0}) {
            state.phase = battle::ecs::AttackPhase::Windup;
            state.phase_remaining = attack.windup_seconds;
        } else {
            enter_active(attack, state);
        }
    }

    void advance_attack_state(const battle::ecs::AttackDefinition& attack, battle::ecs::AttackState& state,
                              battle::ecs::DeltaTime delta_time) {
        switch (state.phase) {
        case battle::ecs::AttackPhase::Idle: {
            break;
        }

        case battle::ecs::AttackPhase::Windup: {
            state.phase_remaining -= delta_time;
            if (state.phase_remaining <= battle::ecs::DeltaTime{0}) {
                enter_active(attack, state);
            }
            break;
        }
        case battle::ecs::AttackPhase::Active: {
            state.phase_remaining -= delta_time;
            if (state.phase_remaining <= battle::ecs::DeltaTime{0}) {
                enter_recovery_or_idle(attack, state);
            }
            break;
        }
        case battle::ecs::AttackPhase::Recovery: {
            state.phase_remaining -= delta_time;
            if (state.phase_remaining <= battle::ecs::DeltaTime{0}) {
                finish_attack(state);
            }
            break;
        }
        }
    }
}

void battle::ecs::attack_resolve_system(World& world, DeltaTime delta_time) {
    for (auto entity : world.registry().pool<AttackRequest>().entities()) {
        auto request = world.registry().try_get<AttackRequest>(entity);
        const auto attack = world.registry().try_get<AttackDefinition>(entity);
        auto cooldown = world.registry().try_get<AttackCooldown>(entity);
        auto transform = world.registry().try_get<Transform>(entity);
        auto state = world.registry().try_get<AttackState>(entity);

        if (!request || !attack || !cooldown || !transform || !state) {
            continue;
        }

        tick_attack_cooldown(*cooldown, delta_time);
        advance_attack_state(*attack, *state, delta_time);

        if (!request->requested) {
            continue;
        }
        request->requested = false;

        if (state->phase != AttackPhase::Idle) {
            continue;
        }
        if (cooldown->remaining_seconds > DeltaTime{0}) {
            continue;
        }

        begin_attack(world, entity, *attack, *transform, *cooldown, *state);
    }
}
