#include "application.hpp"
#include "platform/command_line.hpp"
#include "platform/config.hpp"
#include "platform/logging.hpp"
#include "platform/stop_signal.hpp"

#include <exception>
#include <iostream>
#include <string>

#include "spdlog/spdlog.h"

namespace {
    class LoggingGuard {
    public:
        ~LoggingGuard() {
            battle::ShutdownLogging();
        }
    };
}

int main(int argc, char* argv[]) {
    auto config = battle::DefaultConfig();
    const auto command_line_result = battle::ParseCommandLine(argc, argv, config);
    if (command_line_result != battle::CommandLineResult::Ok) {
        return command_line_result == battle::CommandLineResult::Help ? 0 : 1;
    }

    std::string error;
    if (!battle::InitializeLogging(config.node_name, config.log_level, config.log_mode, error)) {
        std::cerr << "failed to initialize logging: " << error << '\n';
        return 1;
    }
    LoggingGuard logging_guard;

    try {
        battle::StopSignal stop_signal;
        battle::BattleApplication application{config};
        if (!application.run(stop_signal, error)) {
            SPDLOG_CRITICAL("{}", error);
            return 1;
        }
        return 0;
    } catch (const std::exception& exception) {
        SPDLOG_CRITICAL("battle-server failed: {}", exception.what());
        return 1;
    }
}
