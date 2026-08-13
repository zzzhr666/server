#include "logging.hpp"

#include <spdlog/async.h>
#include <spdlog/async_logger.h>
#include <spdlog/sinks/basic_file_sink.h>
#include <spdlog/sinks/stdout_sinks.h>
#include <spdlog/pattern_formatter.h>
#include <spdlog/spdlog.h>

#include <algorithm>
#include <cctype>
#include <chrono>
#include <filesystem>
#include <memory>
#include <string>
#include <vector>

namespace {
    class level_flag_formatter final : public spdlog::custom_flag_formatter {
    public:
        void format(const spdlog::details::log_msg& message,
                    const std::tm&,
                    spdlog::memory_buf_t& destination) override {
            std::string_view level = "unknown";
            switch (message.level) {
            case spdlog::level::trace:
                level = "trace";
                break;
            case spdlog::level::debug:
                level = "debug";
                break;
            case spdlog::level::info:
                level = "info";
                break;
            case spdlog::level::warn:
                level = "warn";
                break;
            case spdlog::level::err:
                level = "error";
                break;
            case spdlog::level::critical:
                level = "fatal";
                break;
            default:
                break;
            }
            destination.append(level.data(), level.data() + level.size());
        }

        std::unique_ptr<spdlog::custom_flag_formatter> clone() const override {
            return std::make_unique<level_flag_formatter>();
        }
    };

    std::string normalize(std::string_view value) {
		auto first = value.find_first_not_of(" \t\r\n");
		if (first == std::string_view::npos) {
			return {};
		}
		auto last = value.find_last_not_of(" \t\r\n");
		std::string result{value.substr(first, last - first + 1)};
        std::transform(result.begin(), result.end(), result.begin(), [](unsigned char character) {
            return static_cast<char>(std::tolower(character));
        });
        return result;
    }

    spdlog::level::level_enum parse_level(std::string_view value, std::string& error) {
        const auto normalized = normalize(value);
        if (normalized == "trace") return spdlog::level::trace;
        if (normalized == "debug") return spdlog::level::debug;
        if (normalized == "info") return spdlog::level::info;
        if (normalized == "warn" || normalized == "warning") return spdlog::level::warn;
        if (normalized == "error") return spdlog::level::err;
        if (normalized == "fatal" || normalized == "critical") return spdlog::level::critical;
        error = "unsupported LOG_LEVEL: " + std::string{value};
        return spdlog::level::info;
    }
}

std::shared_ptr<spdlog::logger> battle::InitializeLogging(
    std::string_view service_name,
    std::string_view level,
    std::string_view mode,
    std::string& error) {
    error.clear();
    if (service_name.empty()) {
        error = "service name is empty";
        return nullptr;
    }

    const auto normalized_mode = normalize(mode);
    if (normalized_mode != "debug" && normalized_mode != "release") {
        error = "unsupported LOG_MODE: " + std::string{mode};
        return nullptr;
    }

    const auto parsed_level = parse_level(level, error);
    if (!error.empty()) {
        return nullptr;
    }

    try {
        std::filesystem::create_directories("./logs");
        auto file_sink = std::make_shared<spdlog::sinks::basic_file_sink_mt>(
            "./logs/" + std::string{service_name} + ".log", true);
        std::vector<spdlog::sink_ptr> sinks;
        if (normalized_mode == "debug") {
            sinks.push_back(std::make_shared<spdlog::sinks::stdout_sink_mt>());
        }
        sinks.push_back(file_sink);

        spdlog::init_thread_pool(8192, 1);
        auto logger = std::make_shared<spdlog::async_logger>(
            "battle",
            sinks.begin(),
            sinks.end(),
            spdlog::thread_pool(),
            spdlog::async_overflow_policy::overrun_oldest);
        logger->set_level(parsed_level);
        auto formatter = std::make_unique<spdlog::pattern_formatter>();
        formatter->add_flag<level_flag_formatter>('q');
        formatter->set_pattern("[%Y/%m/%d][%H:%M:%S.%e][%q][%s:%!:%#] %v");
        logger->set_formatter(std::move(formatter));
        logger->flush_on(spdlog::level::err);
        spdlog::set_default_logger(logger);
        spdlog::flush_every(std::chrono::seconds{1});
        return logger;
    } catch (const std::exception& exception) {
        error = exception.what();
        spdlog::shutdown();
        return nullptr;
    }
}

void battle::ShutdownLogging() {
    spdlog::shutdown();
}
