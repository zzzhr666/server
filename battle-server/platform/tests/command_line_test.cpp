#include "platform/command_line.hpp"

#include "platform/config.hpp"

#include <gtest/gtest.h>

#include <initializer_list>
#include <vector>

namespace {
    battle::CommandLineResult parse(std::initializer_list<const char*> arguments, battle::Config& config) {
        std::vector<char*> argv;
        argv.reserve(arguments.size());
        for (const auto* argument : arguments) {
            argv.push_back(const_cast<char*>(argument));
        }
        return battle::ParseCommandLine(static_cast<int>(argv.size()), argv.data(), config);
    }
}

TEST(CommandLineTest, OverridesMetricsAddress) {
    auto config = battle::DefaultConfig();

    const auto result = parse({"battle_server", "--metrics-addr", "127.0.0.1:9300"}, config);

    EXPECT_EQ(result, battle::CommandLineResult::Ok);
    EXPECT_EQ(config.metrics_addr, "127.0.0.1:9300");
}

TEST(CommandLineTest, RejectsMissingMetricsAddress) {
    auto config = battle::DefaultConfig();

    EXPECT_EQ(parse({"battle_server", "--metrics-addr"}, config), battle::CommandLineResult::Error);
}

TEST(CommandLineTest, RejectsInvalidPlayerCapacity) {
    auto config = battle::DefaultConfig();

    EXPECT_EQ(parse({"battle_server", "--max-players", "0"}, config), battle::CommandLineResult::Error);
}

TEST(CommandLineTest, OverridesRCenterRpcTimeouts) {
    auto config = battle::DefaultConfig();

    const auto result = parse({"battle_server",
                               "--rcenter-register-timeout-seconds", "20",
                               "--rcenter-finish-timeout-seconds", "5"}, config);

    EXPECT_EQ(result, battle::CommandLineResult::Ok);
    EXPECT_EQ(config.rcenter_register_timeout_seconds, std::chrono::seconds{20});
    EXPECT_EQ(config.rcenter_finish_timeout_seconds, std::chrono::seconds{5});
}

TEST(CommandLineTest, RejectsInvalidRCenterRpcTimeouts) {
    auto config = battle::DefaultConfig();

    EXPECT_EQ(parse({"battle_server", "--rcenter-register-timeout-seconds", "0"}, config),
              battle::CommandLineResult::Error);
    EXPECT_EQ(parse({"battle_server", "--rcenter-finish-timeout-seconds", "invalid"}, config),
              battle::CommandLineResult::Error);
}
