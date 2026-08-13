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
        explicit MetricsServer(const std::string& bind_address);
        ~MetricsServer();
        MetricsServer(const MetricsServer&) = delete;
        MetricsServer& operator=(const MetricsServer&) = delete;
        MetricsServer(MetricsServer&&) = delete;
        MetricsServer& operator=(MetricsServer&&) = delete;

        prometheus::Registry& registry() noexcept;

    private:
        std::shared_ptr<prometheus::Registry> registry_;
        std::unique_ptr<prometheus::Exposer> exposer_;
    };
}
