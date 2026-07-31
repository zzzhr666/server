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

        const T& get(Entity entity) const {
            assert(has(entity));
            return dense_components_[sparse_[entity.index]];
        }

        T& get(Entity entity) {
            assert(has(entity));
            return dense_components_[sparse_[entity.index]];
        }

        T* try_get(Entity entity) {
            if (!has(entity))
                return nullptr;

            return &dense_components_[sparse_[entity.index]];
        }

        const T* try_get(Entity entity) const {
            if (!has(entity))
                return nullptr;

            return &dense_components_[sparse_[entity.index]];
        }

        template <typename... Args>
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

        [[nodiscard]] bool empty() const {
            return size() == 0;
        }

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

        void clear() {
            sparse_.clear();
            dense_entities_.clear();
            dense_components_.clear();
        }

        [[nodiscard]] std::size_t size() const {
            return dense_entities_.size();
        }

        [[nodiscard]] const std::vector<Entity>& entities() const {
            return dense_entities_;
        }

        std::vector<T>& components() {
            return dense_components_;
        }

        const std::vector<T>& components() const {
            return dense_components_;
        }

    private:
        std::vector<Entity> dense_entities_;
        std::vector<T> dense_components_;
        std::vector<std::size_t> sparse_;
    };
}
