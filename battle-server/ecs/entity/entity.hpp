#pragma once
#include <cstdint>
#include <compare>
#include <functional>
#include <limits>


namespace battle::ecs {
    using EntityIndex = std::uint32_t;
    using EntityGeneration = std::uint32_t;
    inline constexpr EntityIndex NullEntityIndex = std::numeric_limits<EntityIndex>::max();

    struct Entity {
        EntityIndex index{NullEntityIndex};
        EntityGeneration generation{};

        /// @brief 返回实体是否包含有效索引。
        [[nodiscard]] constexpr bool valid() const noexcept {
            return index != NullEntityIndex;
        }

        /// @brief 允许在条件表达式中检查实体有效性。
        explicit constexpr operator bool() const noexcept {
            return valid();
        }

        /// @brief 按索引和代际比较实体标识。
        auto operator<=>(const Entity&) const = default;

        /// @brief 将实体索引和代际打包为稳定的 64 位值。
        [[nodiscard]] constexpr std::uint64_t packed() const noexcept {
            return static_cast<std::uint64_t>(generation) << 32 | static_cast<std::uint64_t>(index);
        }
    };

    inline constexpr Entity NullEntity{};
}

namespace std { //NOLINT
    template <>
    struct hash<battle::ecs::Entity> {
        /// @brief 使用实体打包值计算哈希。
        size_t operator()(const battle::ecs::Entity& entity) const noexcept {
            return hash<std::uint64_t>{}(entity.packed());
        }
    };
}
