#pragma once

#include <tuple>
#include <type_traits>
#include <utility>

#include "component/component_pool.hpp"
#include "entity/entity_manager.hpp"

namespace battle::ecs {
    template <typename... Components>
    class Registry {
    public:
        template <typename T>
        static constexpr bool RegisteredComponent = (std::is_same_v<T, Components> || ...);

        template <typename T>
        [[nodiscard]] ComponentPool<T>& pool() {
            static_assert(RegisteredComponent<T>);
            return std::get<ComponentPool<T>>(pools_);
        }

        template <typename T>
        [[nodiscard]] const ComponentPool<T>& pool() const {
            static_assert(RegisteredComponent<T>);
            return std::get<ComponentPool<T>>(pools_);
        }

        [[nodiscard]] Entity create() {
            return entity_manager_.create();
        }

        bool destroy(Entity entity) {
            if (!entity_manager_.has(entity)) {
                return false;
            }
            remove_all_components(entity);
            return entity_manager_.destroy(entity);
        }

        [[nodiscard]] bool valid(Entity entity) const {
            return entity_manager_.has(entity);
        }

        [[nodiscard]] const std::vector<Entity>& entities() const {
            return entity_manager_.entities();
        }

        [[nodiscard]] std::size_t size() const {
            return entity_manager_.size();
        }

        template <typename T, typename... Args>
        T& emplace(Entity entity, Args&&... args) {
            assert(valid(entity));
            return pool<T>().emplace(entity, std::forward<Args>(args)...);
        }

        template <typename T>
        bool remove(Entity entity) {
            return pool<T>().remove(entity);
        }

        template <typename T>
        [[nodiscard]] bool has(Entity entity) const {
            return valid(entity) && pool<T>().has(entity);
        }

        template <typename T>
        T& get(Entity entity) {
            return pool<T>().get(entity);
        }

        template <typename T>
        const T& get(Entity entity) const {
            return pool<T>().get(entity);
        }

        template <typename T>
        T* try_get(Entity entity) {
            return pool<T>().try_get(entity);
        }

        template <typename T>
        const T* try_get(Entity entity) const {
            return pool<T>().try_get(entity);
        }

    private:
        void remove_all_components(Entity entity) {
            std::apply([entity](auto&... pools) {
                (pools.remove(entity), ...);
            }, pools_);
        }

    private:
        EntityManager entity_manager_;
        std::tuple<ComponentPool<Components>...> pools_;
    };
}
