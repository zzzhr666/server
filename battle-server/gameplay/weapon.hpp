#pragma once
#include <optional>
#include <string>
#include <string_view>

#include "ecs/component/components.hpp"


namespace battle {
    enum class WeaponKind {
        Sword,
        Dagger,
        Axe,
        Bow
    };

    struct WeaponDefinition {
        WeaponKind kind;
        ecs::AttackDefinition attack;
    };

    WeaponDefinition weapon_definition(WeaponKind kind);

    std::optional<WeaponKind> weapon_kind_from_string(std::string_view value);

    std::string weapon_kind_to_string(WeaponKind kind);
}
