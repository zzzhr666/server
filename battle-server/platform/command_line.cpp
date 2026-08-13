#include "command_line.hpp"

#include "config.hpp"

#include <charconv>
#include <iostream>
#include <string_view>

namespace {
    void print_usage(std::ostream& out, std::string_view program) {
        out << "Usage: " << program << " [options]\n"
            << "  --node-name <name>           Unique battle node name\n"
            << "  --control-addr <host:port>   BattleControl gRPC listen address\n"
            << "  --udp-bind-addr <host:port>  UDP listen address\n"
            << "  --udp-addr <host:port>       UDP address registered for clients\n"
            << "  --rcenter-addr <host:port>   rcenter gRPC address\n"
            << "  --metrics-addr <host:port>   Prometheus metrics listen address\n"
            << "  --max-players <positive-int> Reserved player capacity\n"
            << "  --help                       Show this help\n";
    }

    bool parse_positive_int(std::string_view value, int& result) {
        const auto [end, error] = std::from_chars(value.data(), value.data() + value.size(), result);
        return error == std::errc{} && end == value.data() + value.size() && result > 0;
    }
}

battle::CommandLineResult battle::ParseCommandLine(int argc, char* argv[], Config& config) {
    const auto program = argc > 0 ? std::string_view(argv[0]) : "battle_server";
    for (int i = 1; i < argc; ++i) {
        const auto option = std::string_view(argv[i]);
        if (option == "--help") {
            print_usage(std::cout, program);
            return CommandLineResult::Help;
        }
        if (i + 1 >= argc) {
            std::cerr << "missing value for " << option << '\n';
            print_usage(std::cerr, program);
            return CommandLineResult::Error;
        }

        const auto value = std::string_view(argv[++i]);
        if (option == "--node-name") {
            config.node_name = value;
        } else if (option == "--control-addr") {
            config.control_addr = value;
        } else if (option == "--udp-bind-addr") {
            config.udp_bind_addr = value;
        } else if (option == "--udp-addr") {
            config.udp_addr = value;
        } else if (option == "--rcenter-addr") {
            config.rcenter_addr = value;
        } else if (option == "--metrics-addr") {
            config.metrics_addr = value;
        } else if (option == "--max-players") {
            if (!parse_positive_int(value, config.max_players)) {
                std::cerr << "invalid --max-players value: " << value << '\n';
                return CommandLineResult::Error;
            }
        } else {
            std::cerr << "unknown option: " << option << '\n';
            print_usage(std::cerr, program);
            return CommandLineResult::Error;
        }
    }
    return CommandLineResult::Ok;
}
