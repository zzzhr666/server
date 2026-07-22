#include "control/control_handler.hpp"
#include "control/grpc_server.hpp"
#include "game/game_manager.hpp"
#include "platform/config.hpp"

#include <grpcpp/grpcpp.h>
#include <iostream>
#include <memory>

int main() {
    auto config = battle::DefaultConfig();
    battle::RoomManager room_manager{};

    battle::ControlHandler control_handler{room_manager};

    battle::BattleControlServiceImpl service{control_handler};

    grpc::ServerBuilder builder;

    builder.AddListeningPort(config.control_addr, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);

    auto server = std::unique_ptr<grpc::Server>(builder.BuildAndStart());
    if (!server) {
        std::cerr << "failed to start battle control server on " << config.control_addr << std::endl;
        return 1;
    }

    std::cerr << "battle control server listening on: " << config.control_addr
              << "\nnode = " << config.node_name
              << "\nkcp = " << config.kcp_addr << std::endl;
    server->Wait();
    return 0;
}
