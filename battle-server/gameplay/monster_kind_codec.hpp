#pragma once

#include <string>

#include "monster_kind.hpp"

namespace battle {
    /// @brief 将怪物类型转换为结算协议使用的稳定文本。
    std::string monster_kind_to_string(MonsterKind kind);
}
