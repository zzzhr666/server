#pragma once

#include <cstdint>

namespace battle {
    enum class TrapKind : std::uint8_t {
        Spikes,
        PoisonPool,
        Swamp,
    };
}
