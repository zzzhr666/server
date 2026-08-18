#include "rcenter_client.hpp"

#include <chrono>
#include <utility>

#include "game/game_manager.hpp"
#include "gameplay/monster_kind_codec.hpp"
#include "runtime/battle_runtime.hpp"
#include "spdlog/spdlog.h"

namespace {
    constexpr int kFinishMatchMaxAttempts = 3;

    bool is_retryable_finish_status(const grpc::StatusCode code) {
        switch (code) {
            case grpc::StatusCode::ABORTED:
            case grpc::StatusCode::DEADLINE_EXCEEDED:
            case grpc::StatusCode::INTERNAL:
            case grpc::StatusCode::RESOURCE_EXHAUSTED:
            case grpc::StatusCode::UNAVAILABLE:
                return true;
            default:
                return false;
        }
    }
}

battle::RCenterClient::RCenterClient(std::shared_ptr<grpc::Channel> channel,
                                     const std::chrono::seconds register_timeout,
                                     const std::chrono::seconds finish_timeout)
    : stub_(rcenter::v1::RCenterService::NewStub(std::move(channel))),
      register_timeout_(register_timeout),
      finish_timeout_(finish_timeout) {}

battle::RegisterBattleNodeResult battle::RCenterClient::register_battle_node(
    const Config& config, const RoomManager& room_manager) const {
    rcenter::v1::RegisterBattleNodeRequest request;
    auto node = request.mutable_node();
    node->set_name(config.node_name);
    node->set_udp_addr(config.udp_addr);
    node->set_control_addr(config.control_addr);
    node->set_max_players(config.max_players);
    // 容量来自 RoomManager 的预留数，而不是当前 UDP 已连接数。这样等待重连的玩家
    // 仍占用房间配额，不会在同一节点被超额调度。
    node->set_active_players(static_cast<std::int32_t>(room_manager.active_players()));
    grpc::ClientContext ctx;
    // 首次注册包含容器 DNS、TCP 和 HTTP/2 建连，不能复用结算 RPC 的短超时。
    ctx.set_deadline(std::chrono::system_clock::now() + register_timeout_);
    rcenter::v1::RegisterBattleNodeResponse response;
    grpc::Status status = stub_->RegisterBattleNode(&ctx, request, &response);

    if (!status.ok()) {
        SPDLOG_ERROR("register battle node RPC failed: {}", status.error_message());
        return RegisterBattleNodeResult{.ok = false, .message = status.error_message()};
    }
    SPDLOG_DEBUG("battle node registration heartbeat succeeded");
    return RegisterBattleNodeResult{.ok = true, .message = "registered"};
}

battle::FinishMatchResult battle::RCenterClient::finish_match(const FinishedBattle& finished) const {
    rcenter::v1::FinishMatchRequest request;
    if (finished.player_ids.empty() || finished.reason.empty() || finished.room_name.empty()) {
        return FinishMatchResult{.ok = false, .message = "invalid finish request"};
    }
    for (const auto& player_id : finished.player_ids) {
        request.add_player_ids(player_id);
    }
    request.set_reason(finished.reason);
    request.set_room_name(finished.room_name);
    request.set_combat_duration_ms(finished.settlement.combat_duration_ms);
    // 非胜负结束（如全员断线）仅请求 rcenter 释放匹配上下文，不能附带奖励统计；
    // 胜利或失败必须带完整统计，供 rcenter 以权威规则计算金币。
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

    grpc::Status last_status;
    int attempts = 0;
    for (int attempt = 1; attempt <= kFinishMatchMaxAttempts; ++attempt) {
        attempts = attempt;
        grpc::ClientContext ctx;
        ctx.set_deadline(std::chrono::system_clock::now() + finish_timeout_);
        rcenter::v1::FinishMatchResponse response;
        last_status = stub_->FinishMatch(&ctx, request, &response);
        if (last_status.ok()) {
            SPDLOG_INFO("finish match RPC succeeded room={} players={} reason={} attempt={}",
                        finished.room_name, finished.player_ids.size(), finished.reason, attempt);
            return FinishMatchResult{.ok = true, .message = "finished"};
        }
        if (!is_retryable_finish_status(last_status.error_code()) || attempt == kFinishMatchMaxAttempts) {
            break;
        }
        // room_name 是幂等结算 ID；响应丢失后重试不会重复发放奖励。
        SPDLOG_WARN("finish match RPC retry room={} attempt={} error={}",
                    finished.room_name, attempt, last_status.error_message());
    }
    SPDLOG_ERROR("finish match RPC failed room={} attempts={} error={}",
                 finished.room_name, attempts, last_status.error_message());
    return FinishMatchResult{.ok = false, .message = last_status.error_message()};
}
