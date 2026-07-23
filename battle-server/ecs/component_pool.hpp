#pragma once
#include <cassert>
#include <vector>
#include <utility>
#include <limits>

#include "entity.hpp"


namespace battle::ecs {
    template <typename T>
    class ComponentPool {
    public:
        static constexpr std::size_t INVALID_INDEX = std::numeric_limits<std::size_t>::max();

        bool has(Entity entity) const {
            if (entity >= sparse_.size())
                return false;

            auto index = sparse_[entity];

            if (index == INVALID_INDEX)
                return false;

            if (index >= dense_entities_.size()) {
                return false;
            }

            return dense_entities_[index] == entity;
        }

        const T& get(Entity entity) const {
            assert(has(entity));
            return dense_components_[sparse_[entity]];
        }

        T& get(Entity entity) {
            assert(has(entity));
            return dense_components_[sparse_[entity]];
        }

        T* try_get(Entity entity) {
            if (!has(entity))
                return nullptr;

            return &dense_components_[sparse_[entity]];
        }

        const T* try_get(Entity entity) const {
            if (!has(entity))
                return nullptr;

            return &dense_components_[sparse_[entity]];
        }

        template <typename... Args>
        T& emplace(Entity entity, Args&&... args) {
            if (entity >= sparse_.size()) {
                sparse_.resize(entity + 1, INVALID_INDEX);
            }
            if (sparse_[entity] != INVALID_INDEX) {
                dense_components_[sparse_[entity]] = T{std::forward<Args>(args)...};
                return dense_components_[sparse_[entity]];
            }
            sparse_[entity] = dense_entities_.size();
            dense_entities_.emplace_back(entity);
            dense_components_.emplace_back(std::forward<Args>(args)...);
            return dense_components_.back();
        }

        bool empty() const {
            return size() == 0;
        }

        bool remove(Entity entity) {
            if (!has(entity)) {
                return false;
            }
            auto remove_index = sparse_[entity];
            auto last_index = dense_entities_.size() - 1;
            if (remove_index != last_index) {
                dense_entities_[remove_index] = dense_entities_.back();
                dense_components_[remove_index] = std::move(dense_components_.back());
                sparse_[dense_entities_[remove_index]] = remove_index;
            }
            dense_entities_.pop_back();
            dense_components_.pop_back();
            sparse_[entity] = INVALID_INDEX;
            return true;
        }

        void clear() {
            sparse_.clear();
            dense_entities_.clear();
            dense_components_.clear();
        }

        std::size_t size() const {
            return dense_entities_.size();
        }

        const std::vector<Entity>& entities() const {
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
