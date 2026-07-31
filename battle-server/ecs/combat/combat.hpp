#pragma once

#include <cstdint>
#include <memory>

#include "ecs/entity/entity.hpp"

namespace battle::ecs {
    using CombatActionID = std::uint64_t;
    using CombatEffectID = std::uint64_t;
    inline constexpr CombatActionID InvalidActionID = 0;
    inline constexpr CombatEffectID InvalidEffectID = 0;

    enum class CombatProc : std::uint32_t {
        ChainLightning = 1u << 0,
    };

    using CombatProcMask = std::uint32_t;

    constexpr CombatProcMask combat_proc_mask(CombatProc proc) {
        return static_cast<CombatProcMask>(proc);
    }

    struct CombatActionState {
        CombatActionID action_id{};
        CombatProcMask triggered_procs{};

        [[nodiscard]] bool try_trigger(CombatProc proc) {
            const auto mask = combat_proc_mask(proc);
            if ((triggered_procs & mask) != 0) {
                return false;
            }
            triggered_procs |= mask;
            return true;
        }

        [[nodiscard]] bool has_triggered(CombatProc proc)const {
            return (triggered_procs & combat_proc_mask(proc)) != 0;
        }
    };

    struct CombatContext {
        Entity owner;
        Entity emitter;
        std::shared_ptr<CombatActionState> action_state{};
        CombatEffectID effect_id;
    };
}
