#pragma once
#include <cassert>
#include <vector>
#include <utility>
#include <limits>

#include "ecs/entity/entity.hpp"


namespace battle::ecs {
    template <typename T>
    class ComponentPool {
    public:
        static constexpr std::size_t InvalidIndex = std::numeric_limits<std::size_t>::max();

        /// @brief 返回指定实体是否拥有当前类型组件。
        [[nodiscard]] bool has(Entity entity) const {
            if (!entity || entity.index >= sparse_.size()) {
                return false;
            }
            const auto dense_index = sparse_[entity.index];
            if (dense_index == InvalidIndex || dense_index >= dense_entities_.size()) {
                return false;
            }
            return dense_entities_[dense_index] == entity;
        }

        /// @brief 返回指定实体组件的只读引用，实体必须已拥有该组件。
        const T& get(Entity entity) const {
            assert(has(entity));
            return dense_components_[sparse_[entity.index]];
        }

        /// @brief 返回指定实体组件的可写引用，实体必须已拥有该组件。
        T& get(Entity entity) {
            assert(has(entity));
            return dense_components_[sparse_[entity.index]];
        }

        /// @brief 返回指定实体组件的可写指针，不存在时返回 nullptr。
        T* try_get(Entity entity) {
            if (!has(entity))
                return nullptr;

            return &dense_components_[sparse_[entity.index]];
        }

        /// @brief 返回指定实体组件的只读指针，不存在时返回 nullptr。
        const T* try_get(Entity entity) const {
            if (!has(entity))
                return nullptr;

            return &dense_components_[sparse_[entity.index]];
        }

        template <typename... Args>
        /// @brief 创建或替换实体组件，并维护稀疏索引与紧凑数组映射。
        T& emplace(Entity entity, Args&&... args) {
            assert(entity);
            if (entity.index >= sparse_.size()) {
                sparse_.resize(static_cast<std::size_t>(entity.index) + 1, InvalidIndex);
            }
            std::size_t index = sparse_[entity.index];
            if (index == InvalidIndex) {
                const auto new_index = dense_entities_.size();
                dense_entities_.emplace_back(entity);
                dense_components_.emplace_back(std::forward<Args>(args)...);
                sparse_[entity.index] = new_index;
                return dense_components_.back();
            }

            dense_components_[index] = T(std::forward<Args>(args)...);
            dense_entities_[index] = entity;
            return dense_components_[index];
        }

        /// @brief 返回组件池是否为空。
        [[nodiscard]] bool empty() const {
            return size() == 0;
        }

        /// @brief 删除实体组件，并用尾部元素填补紧凑数组空位。
        bool remove(Entity entity) {
            if (!has(entity)) {
                return false;
            }
            auto remove_index = sparse_[entity.index];
            auto last_index = dense_entities_.size() - 1;
            if (remove_index != last_index) {
                const Entity moved_entity = dense_entities_.back();
                dense_entities_[remove_index] = moved_entity;
                dense_components_[remove_index] = std::move(dense_components_.back());
                sparse_[moved_entity.index] = remove_index;
            }
            dense_entities_.pop_back();
            dense_components_.pop_back();
            sparse_[entity.index] = InvalidIndex;
            return true;
        }

        /// @brief 清空组件数据及其实体索引。
        void clear() {
            sparse_.clear();
            dense_entities_.clear();
            dense_components_.clear();
        }

        /// @brief 返回组件数量。
        [[nodiscard]] std::size_t size() const {
            return dense_entities_.size();
        }

        /// @brief 返回与组件紧凑数组同序的实体列表。
        [[nodiscard]] const std::vector<Entity>& entities() const {
            return dense_entities_;
        }

        /// @brief 返回紧凑组件数组的可写引用。
        std::vector<T>& components() {
            return dense_components_;
        }

        /// @brief 返回紧凑组件数组的只读引用。
        const std::vector<T>& components() const {
            return dense_components_;
        }

    private:
        std::vector<Entity> dense_entities_;
        std::vector<T> dense_components_;
        std::vector<std::size_t> sparse_;
    };
}
