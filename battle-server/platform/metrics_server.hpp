#pragma once

#include <memory>
#include <string>

namespace prometheus {
    class Exposer;
    class Registry;
}


namespace battle {
    class MetricsServer {
    public:
        /// @brief 绑定地址并启动 Prometheus 指标服务。
        explicit MetricsServer(const std::string& bind_address);
        /// @brief 停止指标服务并释放监听资源。
        ~MetricsServer();
        /// @brief MetricsServer 独占监听资源，不允许复制。
        MetricsServer(const MetricsServer&) = delete;
        /// @brief MetricsServer 独占监听资源，不允许复制赋值。
        MetricsServer& operator=(const MetricsServer&) = delete;
        /// @brief MetricsServer 的后台服务不可移动。
        MetricsServer(MetricsServer&&) = delete;
        /// @brief MetricsServer 的后台服务不可移动赋值。
        MetricsServer& operator=(MetricsServer&&) = delete;

        /// @brief 返回供业务指标注册使用的 Prometheus 注册表。
        prometheus::Registry& registry() noexcept;

    private:
        std::shared_ptr<prometheus::Registry> registry_;
        std::unique_ptr<prometheus::Exposer> exposer_;
    };
}
