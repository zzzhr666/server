#pragma once

#include <vector>
#include <limits>
#include <cstddef>

#include "entity.hpp"

namespace battle::ecs {
    inline constexpr std::size_t InvalidDenseIndex = std::numeric_limits<std::size_t>::max();


    class EntityManager {
    public:
        /// @brief 创建空的实体管理器。
        EntityManager() = default;

        /// @brief 校验实体索引、代际和存活状态是否仍匹配。
        [[nodiscard]] bool has(Entity entity) const {
            if (!entity || entity.index >= slots_.size()) {
                return false;
            }
            const auto& slot = slots_[entity.index];
            return slot.alive && slot.generation == entity.generation;
        }

        /// @brief 返回当前存活实体的紧凑列表。
        [[nodiscard]] const std::vector<Entity>& entities() const {
            return alive_entities_;
        }

        /// @brief 销毁有效实体，递增代际并回收其索引。
        bool destroy(Entity entity) {
            if (!has(entity)) {
                return false;
            }
            auto& slot = slots_[entity.index];
            auto remove_index = slot.dense_index;
            auto last_index = alive_entities_.size() - 1;
            if (remove_index != last_index) {
                alive_entities_[remove_index] = alive_entities_[last_index];
                slots_[alive_entities_[last_index].index].dense_index = remove_index;
            }
            alive_entities_.pop_back();
            slot.alive = false;
            slot.dense_index = InvalidDenseIndex;
            slot.generation++;
            free_indices_.push_back(entity.index);
            return true;
        }

        /// @brief 分配新实体，优先复用已销毁实体的索引。
        [[nodiscard]] Entity create() {
            EntityIndex index;
            if (!free_indices_.empty()) {
                index = free_indices_.back();
                free_indices_.pop_back();
            } else {
                index = static_cast<EntityIndex>(slots_.size());
                slots_.emplace_back();
            }

            slots_[index].alive = true;
            slots_[index].dense_index = alive_entities_.size();
            Entity entity{
                .index = index,
                .generation = slots_[index].generation
            };
            alive_entities_.push_back(entity);
            return entity;
        }

        /// @brief 返回当前存活实体数量。
        [[nodiscard]] std::size_t size() const {
            return alive_entities_.size();
        }

    private:
        struct EntitySlot {
            EntityGeneration generation = 1;
            std::size_t dense_index = InvalidDenseIndex;
            bool alive = false;
        };

        std::vector<EntitySlot> slots_;
        std::vector<EntityIndex> free_indices_;
        std::vector<Entity> alive_entities_;
    };
}
