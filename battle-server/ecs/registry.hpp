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
        /// @brief 返回指定已注册组件类型的可写组件池。
        [[nodiscard]] ComponentPool<T>& pool() {
            static_assert(RegisteredComponent<T>);
            return std::get<ComponentPool<T>>(pools_);
        }

        template <typename T>
        /// @brief 返回指定已注册组件类型的只读组件池。
        [[nodiscard]] const ComponentPool<T>& pool() const {
            static_assert(RegisteredComponent<T>);
            return std::get<ComponentPool<T>>(pools_);
        }

        /// @brief 创建并返回一个新实体。
        [[nodiscard]] Entity create() {
            return entity_manager_.create();
        }

        /// @brief 删除实体的全部组件后销毁实体标识。
        bool destroy(Entity entity) {
            if (!entity_manager_.has(entity)) {
                return false;
            }
            remove_all_components(entity);
            return entity_manager_.destroy(entity);
        }

        /// @brief 返回实体标识是否仍然存活。
        [[nodiscard]] bool valid(Entity entity) const {
            return entity_manager_.has(entity);
        }

        /// @brief 返回当前存活实体列表。
        [[nodiscard]] const std::vector<Entity>& entities() const {
            return entity_manager_.entities();
        }

        /// @brief 返回当前存活实体数量。
        [[nodiscard]] std::size_t size() const {
            return entity_manager_.size();
        }

        template <typename T, typename... Args>
        /// @brief 为有效实体创建或替换指定组件。
        T& emplace(Entity entity, Args&&... args) {
            assert(valid(entity));
            return pool<T>().emplace(entity, std::forward<Args>(args)...);
        }

        template <typename T>
        /// @brief 从实体移除指定类型组件。
        bool remove(Entity entity) {
            return pool<T>().remove(entity);
        }

        template <typename T>
        /// @brief 返回有效实体是否拥有指定类型组件。
        [[nodiscard]] bool has(Entity entity) const {
            return valid(entity) && pool<T>().has(entity);
        }

        template <typename T>
        /// @brief 返回实体指定组件的可写引用。
        T& get(Entity entity) {
            return pool<T>().get(entity);
        }

        template <typename T>
        /// @brief 返回实体指定组件的只读引用。
        const T& get(Entity entity) const {
            return pool<T>().get(entity);
        }

        template <typename T>
        /// @brief 返回实体指定组件的可写指针，不存在时返回 nullptr。
        T* try_get(Entity entity) {
            return pool<T>().try_get(entity);
        }

        template <typename T>
        /// @brief 返回实体指定组件的只读指针，不存在时返回 nullptr。
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
