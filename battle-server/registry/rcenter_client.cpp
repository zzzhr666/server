#include "rcenter_client.hpp"

#include <utility>

#include "game/game_manager.hpp"
#include "gameplay/monster_kind_codec.hpp"
#include "runtime/battle_runtime.hpp"

battle::RCenterClient::RCenterClient(std::shared_ptr<grpc::Channel> channel)
    : stub_(rcenter::v1::RCenterService::NewStub(std::move(channel))) {}

battle::RegisterBattleNodeResult battle::RCenterClient::register_battle_node(
    const Config& config, const RoomManager& room_manager) const {
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

battle::FinishMatchResult battle::RCenterClient::finish_match(const FinishedBattle& finished) const {
    rcenter::v1::FinishMatchRequest request;
    if (finished.player_ids.empty() || finished.reason.empty()) {
        return FinishMatchResult{.ok = false, .message = "invalid finish request"};
    }
    for (const auto& player_id : finished.player_ids) {
        request.add_player_ids(player_id);
    }
    request.set_reason(finished.reason);

    if (finished.reason == "victory" || finished.reason == "defeat") {
        if (finished.settlement.players.empty()) {
            return FinishMatchResult{.ok = false, .message = "missing settlement players"};
        }
        for (const auto& player : finished.settlement.players) {
            auto stat = request.add_player_stats();
            stat->set_player_id(player.player_id);
            stat->set_total_kills(player.total_kills);
            for (const auto& [monster_kind, count] : player.kills) {
                auto kill_detail = stat->add_kills();
                kill_detail->set_monster_kind(monster_kind_to_string(monster_kind));
                kill_detail->set_count(count);
            }
        }
    }


    grpc::ClientContext ctx;
    rcenter::v1::FinishMatchResponse response;
    grpc::Status status = stub_->FinishMatch(&ctx, request, &response);

    return status.ok()
               ? FinishMatchResult{.ok = true, .message = "finished"}
               : FinishMatchResult{.ok = false, .message = status.error_message()};
}
