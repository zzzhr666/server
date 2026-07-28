#include "monster_kind_codec.hpp"

std::string battle::monster_kind_to_string(MonsterKind kind) {
    switch (kind) {
    case MonsterKind::Melee:
        return "melee";
    default:
        return "unknown";
    }
}
