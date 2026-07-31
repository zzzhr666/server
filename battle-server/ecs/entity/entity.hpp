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

        [[nodiscard]] constexpr bool valid() const noexcept {
            return index != NullEntityIndex;
        }

        explicit constexpr operator bool() const noexcept {
            return valid();
        }

        auto operator<=>(const Entity&) const = default;

        [[nodiscard]] constexpr std::uint64_t packed() const noexcept {
            return static_cast<std::uint64_t>(generation) << 32 | static_cast<std::uint64_t>(index);
        }
    };

    inline constexpr Entity NullEntity{};
}

namespace std {
    template <>
    struct hash<battle::ecs::Entity> {
        size_t operator()(const battle::ecs::Entity& entity) const noexcept {
            return hash<std::uint64_t>{}(entity.packed());
        }
    };
}
