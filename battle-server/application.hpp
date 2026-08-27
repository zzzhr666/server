#pragma once

#include "control/control_handler.hpp"
#include "control/grpc_server.hpp"
#include "game/game_manager.hpp"
#include "net/udp_server.hpp"
#include "platform/config.hpp"
#include "platform/metrics_server.hpp"
#include "registry/node_registrar.hpp"
#include "registry/rcenter_client.hpp"
#include "runtime/battle_runtime.hpp"
#include "session/session_manager.hpp"

#include <memory>
#include <string>

#include <grpcpp/grpcpp.h>

#include "platform/metrics.hpp"

namespace battle {
    class StopSignal;

    /// @brief BattleApplication 组装并管理战斗服全部进程级资源。
    class BattleApplication {
    public:
        /// @brief 使用完整节点配置创建 battle-server 应用。
        explicit BattleApplication(Config config);
        /// @brief 停止服务并释放应用持有的网络与后台资源。
        ~BattleApplication();

        /// @brief BattleApplication 独占服务资源，不允许复制。
        BattleApplication(const BattleApplication&) = delete;
        /// @brief BattleApplication 独占服务资源，不允许复制赋值。
        BattleApplication& operator=(const BattleApplication&) = delete;

        /// @brief 启动所有服务并运行到收到停止信号。
        bool run(const StopSignal& stop_signal, std::string& error);

    private:
        void stop();

        Config config_;
        MetricsServer metrics_server_;
        BattleMetrics battle_metrics_;
        RoomManager room_manager_;
        SessionManager session_manager_;
        UdpServer udp_server_;
        RCenterClient rcenter_client_;
        BattleRuntime battle_runtime_;
        ControlHandler control_handler_;
        BattleControlServiceImpl control_service_;
        NodeRegistrar node_registrar_;
        std::unique_ptr<grpc::Server> grpc_server_;
        bool udp_started_{false};
        bool runtime_started_{false};
        bool registrar_started_{false};
    };
}
