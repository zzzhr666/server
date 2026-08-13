#pragma once

#include <memory>
#include <string_view>

namespace spdlog {
    class logger;
}

namespace battle {
    /// @brief InitializeLogging creates and registers the process-wide async logger.
    std::shared_ptr<spdlog::logger> InitializeLogging(
        std::string_view service_name,
        std::string_view level,
        std::string_view mode,
        std::string& error);

    /// @brief ShutdownLogging flushes and closes the process-wide logger.
    void ShutdownLogging();
}
