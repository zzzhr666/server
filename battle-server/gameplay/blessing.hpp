#pragma once

#include <cstdint>

namespace battle {
    enum class BlessingID : std::uint8_t {
        BurnOnHit = 0,
        LifeSteal,
        FreezeOnHit,
        CriticalStrike,
        ChainLightning,
        Frenzy,
        Swift,
        Toughness,
        HeavyStrike,
        ArmorBreak,
        Revenge,
        SoulHarvest,
    };

    struct PlayerBlessing {
        BlessingID blessing_id = BlessingID::BurnOnHit;
        int level = 1;
    };
}
