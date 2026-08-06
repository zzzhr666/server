#pragma once
#include <functional>
#include <vector>

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    using sysFunc = std::function<void(ecs::World&, DeltaTime)>;
    /// @brief SystemScheduler 按注册顺序执行 ECS 系统；顺序即玩法规则的一部分。
    class SystemScheduler {
    public:
        SystemScheduler();
        SystemScheduler(std::initializer_list<sysFunc> funcs);
        /// @brief 在调度链尾部注册一个系统。
        void add_system(sysFunc func);
        /// @brief 按注册顺序对世界执行一次完整系统链。
        void tick(World& world, DeltaTime delta_time) const;
    private:
        std::vector<sysFunc> systems_;
    };
}
