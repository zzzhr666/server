#include "application.hpp"

#include "platform/stop_signal.hpp"

#include <chrono>
#include <utility>

#include "spdlog/spdlog.h"

battle::BattleApplication::BattleApplication(Config config)
    : config_(std::move(config)),
      metrics_server_(config_.metrics_addr),
      battle_metrics_(metrics_server_.registry()),
      room_manager_(battle_metrics_),
      session_manager_(room_manager_, battle_metrics_),
      udp_server_(config_.udp_bind_addr, session_manager_, battle_metrics_),
      rcenter_client_(grpc::CreateChannel(config_.rcenter_addr, grpc::InsecureChannelCredentials()),
                      config_.rcenter_register_timeout_seconds,
                      config_.rcenter_finish_timeout_seconds),
      battle_runtime_(
          room_manager_,
          session_manager_,
          battle_metrics_,
          [this](const v1::ServerPacket& packet, const UdpEndpoint& endpoint) {
              udp_server_.send_packet(packet, endpoint);
          },
          {},
          [this](const FinishedBattle& finished) {
              const auto result = rcenter_client_.finish_match(finished);
              if (!result.ok) {
                  SPDLOG_ERROR("failed to finish match in rcenter: {}", result.message);
              }
          },
          config_.tick_rate,
          config_.session_idle_timeout_seconds,
          config_.all_players_disconnected_timeout),
      control_handler_(room_manager_, battle_runtime_, battle_metrics_),
      control_service_(control_handler_),
      node_registrar_(config_, rcenter_client_, room_manager_) {
    udp_server_.set_runtime(battle_runtime_);
}

battle::BattleApplication::~BattleApplication() {
    stop();
}

bool battle::BattleApplication::run(const StopSignal& stop_signal, std::string& error) {
    error.clear();
    if (!udp_server_.start()) {
        error = "failed to start battle UDP server on " + config_.udp_bind_addr;
        return false;
    }
    udp_started_ = true;

    battle_runtime_.start();
    runtime_started_ = true;

    grpc::ServerBuilder builder;
    builder.AddListeningPort(config_.control_addr, grpc::InsecureServerCredentials());
    builder.RegisterService(&control_service_);
    grpc_server_ = builder.BuildAndStart();
    if (!grpc_server_) {
        error = "failed to start battle control server on " + config_.control_addr;
        return false;
    }

    const auto register_result = rcenter_client_.register_battle_node(config_, room_manager_);
    if (!register_result.ok) {
        error = "failed to register battle node to rcenter " + config_.rcenter_addr + ": " + register_result.message;
        return false;
    }

    node_registrar_.start();
    registrar_started_ = true;
    SPDLOG_INFO(
        "battle-server listening control={} udp_bind={} udp_public={} metrics={} node={} tick_rate={}",
        config_.control_addr,
        config_.udp_bind_addr,
        config_.udp_addr,
        config_.metrics_addr,
        config_.node_name,
        config_.tick_rate);

    stop_signal.wait();
    SPDLOG_INFO("shutdown signal received");
    stop();
    return true;
}

void battle::BattleApplication::stop() {
    if (registrar_started_) {
        node_registrar_.stop();
        registrar_started_ = false;
    }
    if (grpc_server_) {
        grpc_server_->Shutdown(std::chrono::system_clock::now() + std::chrono::seconds{5});
        grpc_server_->Wait();
        grpc_server_.reset();
    }
    if (runtime_started_) {
        battle_runtime_.stop();
        runtime_started_ = false;
    }
    if (udp_started_) {
        udp_server_.stop();
        udp_started_ = false;
    }
}
