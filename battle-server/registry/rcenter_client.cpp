#include "rcenter_client.hpp"

#include <cstdint>
#include <utility>

#include "game/game_manager.hpp"

battle::RCenterClient::RCenterClient(std::shared_ptr<grpc::Channel> channel)
    : stub_(rcenter::v1::RCenterService::NewStub(std::move(channel))) {}

battle::RegisterBattleNodeResult battle::RCenterClient::register_battle_node(
    const Config& config, const RoomManager& room_manager) {
    rcenter::v1::RegisterBattleNodeRequest request;
    auto node = request.mutable_node();
    node->set_name(config.node_name);
    node->set_kcp_addr(config.kcp_addr);
    node->set_control_addr(config.control_addr);
    node->set_max_players(config.max_players);
    node->set_active_players(static_cast<std::int32_t>(room_manager.active_players()));
    grpc::ClientContext ctx;
    rcenter::v1::RegisterBattleNodeResponse response;
    grpc::Status status = stub_->RegisterBattleNode(&ctx, request, &response);

    return status.ok()
               ? RegisterBattleNodeResult{.ok = true, .message = "registered"}
               : RegisterBattleNodeResult{.ok = false, .message = status.error_message()};
}
