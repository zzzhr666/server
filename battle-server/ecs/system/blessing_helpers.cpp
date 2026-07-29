#include "blessing_helpers.hpp"

#include <algorithm>

#include "ecs/world.hpp"

const battle::ecs::BlessingStack* battle::ecs::find_blessing(const World& world, Entity entity,
                                                             BlessingID blessing_id) {
    const auto* inventory = world.blessing_inventories().try_get(entity);
    if (!inventory) {
        return nullptr;
    }
    auto blessing_it = std::ranges::find_if(inventory->blessings, [blessing_id](const BlessingStack& blessing) {
        return blessing.blessing_id == blessing_id;
    });
    return blessing_it == inventory->blessings.end() ? nullptr : &*blessing_it;
}

int battle::ecs::blessing_level(const World& world, Entity entity, BlessingID blessing_id) {
    const auto* blessing = find_blessing(world, entity, blessing_id);
    return blessing ? blessing->level : 0;
}

bool battle::ecs::roll_percent(World& world, int chance_percent) {
    if (chance_percent <= 0) {
        return false;
    }
    if (chance_percent >= 100) {
        return true;
    }
    return world.random_percent() <= chance_percent;
}
