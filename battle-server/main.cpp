#include "control/control_handler.hpp"
#include "control/grpc_server.hpp"
#include "game/game_manager.hpp"
#include "platform/config.hpp"

#include <grpcpp/grpcpp.h>
#include <iostream>
#include <memory>

#include "net/udp_server.hpp"
#include "registry/node_registrar.hpp"
#include "registry/rcenter_client.hpp"
#include "session/session_manager.hpp"

int main() {
    auto config = battle::DefaultConfig();
    battle::RoomManager room_manager{};

    battle::ControlHandler control_handler{room_manager};

    battle::BattleControlServiceImpl service{control_handler};
    battle::SessionManager session_manager{room_manager};
    battle::UdpServer udp_server{config.kcp_addr, room_manager, session_manager};
    if (!udp_server.start()) {
        std::cerr << "failed to start battle udp server on " << config.kcp_addr << std::endl;
        return 1;
    }

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
    battle::RCenterClient rcenter_client{
        grpc::CreateChannel(config.rcenter_addr, grpc::InsecureChannelCredentials())
    };
    auto register_res = rcenter_client.register_battle_node(config, room_manager);
    if (!register_res.ok) {
        std::cerr << "failed to register battle node to rcenter " << config.rcenter_addr
            << ':' << register_res.message << std::endl;
        udp_server.stop();
        server->Shutdown();
        return 1;
    }
    battle::NodeRegistrar node_registrar{config,rcenter_client,room_manager};
    node_registrar.start();
    server->Wait();
    node_registrar.stop();
    udp_server.stop();
    return 0;
}
