#pragma once

namespace battle {
    enum class MonsterKind {
        Melee,
    };
    struct MonsterDefinition {
        MonsterKind kind;
        int base_health;
        float base_move_speed;
    };

    MonsterDefinition monster_definition(MonsterKind kind);

}