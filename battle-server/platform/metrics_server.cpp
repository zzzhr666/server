#include "metrics_server.hpp"


#include <prometheus/exposer.h>
#include <prometheus/registry.h>

battle::MetricsServer::MetricsServer(const std::string& bind_address)
    : registry_(std::make_shared<prometheus::Registry>()),
      exposer_(std::make_unique<prometheus::Exposer>(bind_address)) {
    exposer_->RegisterCollectable(registry_);
}

battle::MetricsServer::~MetricsServer() = default;

prometheus::Registry& battle::MetricsServer::registry() noexcept {
    return *registry_;
}
