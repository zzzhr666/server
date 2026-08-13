#include "control/control_handler.hpp"
#include "control/grpc_server.hpp"
#include "game/game_manager.hpp"
#include "platform/config.hpp"
#include "platform/logging.hpp"

#include <charconv>
#include <grpcpp/grpcpp.h>
#include <iostream>
#include <memory>
#include <string_view>

#include "net/udp_server.hpp"
#include "registry/node_registrar.hpp"
#include "registry/rcenter_client.hpp"
#include "runtime/battle_runtime.hpp"
#include "session/session_manager.hpp"
#include "spdlog/spdlog.h"

namespace {
    enum class CommandLineResult {
        Ok,
        Help,
        Error,
    };

    void print_usage(std::ostream& out, std::string_view program) {
        out << "Usage: " << program << " [options]\n"
            << "  --node-name <name>           Unique battle node name\n"
            << "  --control-addr <host:port>   BattleControl gRPC listen address\n"
            << "  --udp-bind-addr <host:port>  UDP listen address\n"
            << "  --udp-addr <host:port>       UDP address registered for clients\n"
            << "  --rcenter-addr <host:port>   rcenter gRPC address\n"
            << "  --max-players <positive-int> Reserved player capacity\n"
            << "  --help                       Show this help\n";
    }

    bool parse_positive_int(std::string_view value, int& result) {
        const auto [end, error] = std::from_chars(value.data(), value.data() + value.size(), result);
        return error == std::errc{} && end == value.data() + value.size() && result > 0;
    }

    CommandLineResult parse_command_line(int argc, char* argv[], battle::Config& config) {
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
}

int main(int argc, char* argv[]) {
    auto config = battle::DefaultConfig();
    const auto command_line_result = parse_command_line(argc, argv, config);
    if (command_line_result == CommandLineResult::Help) {
        return 0;
    }
    if (command_line_result == CommandLineResult::Error) {
        return 1;
    }

    std::string logging_error;
    if (!battle::InitializeLogging(config.node_name, config.log_level, config.log_mode, logging_error)) {
        std::cerr << "failed to initialize logging: " << logging_error << '\n';
        return 1;
    }
    struct LoggingGuard {
        ~LoggingGuard() { battle::ShutdownLogging(); }
    } logging_guard;

    SPDLOG_INFO("battle server starting: node={}, control={}, udp_bind={}, udp_public={}",
        config.node_name, config.control_addr, config.udp_bind_addr, config.udp_addr);
    battle::RoomManager room_manager{};
    battle::SessionManager session_manager{room_manager};
    battle::UdpServer udp_server{config.udp_bind_addr, session_manager};
    battle::RCenterClient rcenter_client{
        grpc::CreateChannel(config.rcenter_addr, grpc::InsecureChannelCredentials())
    };
    battle::BattleRuntime battle_runtime{
        room_manager,
        session_manager,
        [&udp_server](const battle::v1::ServerPacket& packet, const battle::UdpEndpoint& endpoint) {
            udp_server.send_packet(packet, endpoint);
        },
        {},
        [&rcenter_client](const battle::FinishedBattle& finished) {
            auto res = rcenter_client.finish_match(finished);
            if (!res.ok) {
                SPDLOG_ERROR("failed to finish match in rcenter: {}", res.message);
            }
        },
        config.tick_rate,
        config.session_idle_timeout_seconds,
        config.all_players_disconnected_timeout
    };
    udp_server.set_runtime(battle_runtime);

    battle::ControlHandler control_handler{room_manager, battle_runtime};
    battle::BattleControlServiceImpl service{control_handler};

    if (!udp_server.start()) {
        SPDLOG_CRITICAL("failed to start battle udp server on {}", config.udp_bind_addr);
        return 1;
    }
    battle_runtime.start();

    grpc::ServerBuilder builder;

    builder.AddListeningPort(config.control_addr, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);

    auto server = std::unique_ptr<grpc::Server>(builder.BuildAndStart());
    if (!server) {
        SPDLOG_CRITICAL("failed to start battle control server on {}", config.control_addr);
        battle_runtime.stop();
        udp_server.stop();
        return 1;
    }

    SPDLOG_INFO("battle control server listening: control={}, node={}, udp_bind={}, udp_public={}, tick_rate={}",
        config.control_addr, config.node_name, config.udp_bind_addr, config.udp_addr, config.tick_rate);

    auto register_res = rcenter_client.register_battle_node(config, room_manager);
    if (!register_res.ok) {
        SPDLOG_CRITICAL("failed to register battle node to rcenter {}: {}", config.rcenter_addr, register_res.message);
        battle_runtime.stop();
        udp_server.stop();
        server->Shutdown();
        return 1;
    }
    battle::NodeRegistrar node_registrar{config, rcenter_client, room_manager};
    node_registrar.start();
    server->Wait();
    node_registrar.stop();
    battle_runtime.stop();
    udp_server.stop();
    return 0;
}
