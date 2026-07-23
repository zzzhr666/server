#pragma once
#include <cstdint>


namespace battle::ecs {
    using Entity = std::uint32_t;
    constexpr Entity INVALID_ENTITY = 0;
}
