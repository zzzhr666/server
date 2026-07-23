#pragma once
#include <algorithm>
#include <vector>

#include "entity.hpp"

namespace battle::ecs {
    class EntityManager {
    public:
        EntityManager() : next_entity_(1) {}

        [[nodiscard]] bool has(Entity entity) const {
            return std::ranges::any_of(alive_entities_, [entity](Entity x) {
                return x == entity;
            });
        }

        [[nodiscard]] const std::vector<Entity>& entities() const {
            return alive_entities_;
        }

        Entity create() {
            auto entity = next_entity_++;
            alive_entities_.push_back(entity);
            return entity;
        }

    private:
        Entity next_entity_;
        std::vector<Entity> alive_entities_;
    };
}
