#pragma once
#include <optional>
#include <string>
#include <string_view>

#include "ecs/component/components.hpp"


namespace battle {
    /// @brief 当前支持的初始英雄类型。
    enum class HeroKind {
        Fire,
        Ice,
        Rock,
        Nature
    };

    /// @brief 英雄类型与其 ECS 攻击定义的对应关系。
    struct HeroDefinition {
        HeroKind kind{};
        ecs::AttackDefinition attack;
    };

    /// @brief 返回指定英雄的权威攻击数值。
    HeroDefinition hero_definition(HeroKind kind);

    /// @brief 将匹配协议中的英雄文本解析为英雄类型。
    std::optional<HeroKind> hero_kind_from_string(std::string_view value);

    /// @brief 将英雄类型转换为协议使用的文本。
    std::string hero_kind_to_string(HeroKind kind);
}
