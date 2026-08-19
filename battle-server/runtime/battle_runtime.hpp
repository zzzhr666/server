#pragma once


#include <functional>
#include <mutex>
#include <string>
#include <memory>
#include <vector>
#include <unordered_map>
#include <unordered_set>
#include <atomic>
#include <chrono>
#include <thread>


#include "ecs/time.hpp"
#include "game/room.hpp"
#include "net/udp_endpoint.hpp"
#include "proto/battle/v1/session.pb.h"
#include "player_input.hpp"
#include "battle_instance.hpp"

namespace battle {
    class SessionManager;
    class RoomManager;
    class BattleInstance;
    class BattleMetrics;

    /// @brief 向指定 UDP 端点发送服务端协议包的回调。
    using SendPacketCallback = std::function<void(const v1::ServerPacket&, const UdpEndpoint&)>;

    /// @brief 描述一场已结束战斗，供 rcenter 释放匹配状态并结算奖励。
    struct FinishedBattle {
        /// @brief 已结束房间及其完整玩家名单。
        std::string room_name;
        std::vector<std::int64_t> player_ids;
        /// @brief 局内结算数据和结束原因文本。
        BattleSettlement settlement;
        std::string reason;
    };

    /// @brief BattleRuntime 管理房间内 BattleInstance 的 tick、广播、断线清理和结束回调。
    class BattleRuntime {
    public:
        /// @brief 为测试或特殊房间创建战斗实例的工厂。
        using BattleInstanceFactory = std::function<std::unique_ptr<BattleInstance>(BattleInstanceConfig)>;
        /// @brief 战斗结束后通知 rcenter 的回调。
        using FinishMatchCallback = std::function<void(const FinishedBattle&)>;
        BattleRuntime(RoomManager& room_manager, SessionManager& session_manager, BattleMetrics& metrics,
                      SendPacketCallback send_packet_callback, BattleInstanceFactory factory = {},
                      FinishMatchCallback finish_match_callback = {}, int tick_rate = 60,
                      std::chrono::seconds session_idle_timeout_seconds = std::chrono::seconds{15},
                      std::chrono::seconds all_players_disconnected_timeout_seconds = std::chrono::seconds{90});
        ~BattleRuntime();

        /// @brief 在完整 roster 加入后创建并启动对应房间的战斗实例。
        void start_room(const std::string& room_name);

        /// @brief 推进所有活跃房间，并广播快照或处理超时结束。
        void tick(ecs::DeltaTime delta_time);

        /// @brief 将玩家输入转交到其房间的战斗实例。
        bool receive_input(const std::string& room_name, std::int64_t player_id, PlayerInput input);

        /// @brief 将奖励选择操作转交到其房间的战斗实例。
        bool choose_blessing(const std::string& room_name, std::int64_t player_id, int option_id);

        /// @brief 启动固定频率的后台 tick 线程。
        void start();

        /// @brief 停止后台 tick 线程。
        void stop();

        /// @brief 按外部原因结束房间并触发资源清理。
        EndRoomResult end_room(const std::string& room_name, const std::string& reason);

    private:
        RoomManager& room_manager_;
        SessionManager& session_manager_;
        BattleMetrics& metrics_;
        SendPacketCallback send_packet_;
        FinishMatchCallback finish_match_callback_;
        std::mutex mutex_;
        /// @brief 以房间名索引的活跃战斗实例。
        std::unordered_map<std::string, std::unique_ptr<BattleInstance>> instances_;
        /// @brief 正在启动中的房间，避免重复 hello 并发创建实例。
        std::unordered_set<std::string> starting_rooms_;
        std::atomic<bool> running_;
        std::thread tick_thread_;
        BattleInstanceFactory instance_factory_;
        /// @brief 规范化后的权威模拟频率，同时用于实例配置和网络快照。
        std::uint32_t tick_rate_;
        /// @brief 每次后台 tick 推进的固定模拟时间，与线程实际唤醒延迟解耦。
        ecs::DeltaTime fixed_delta_time_;
        std::chrono::steady_clock::duration tick_interval_;
        /// @brief 单个 UDP 会话未收到有效数据时的断线阈值。
        std::chrono::seconds session_idle_timeout_;
        std::chrono::seconds all_players_disconnected_timeout_;
        /// @brief 每个全员断线房间首次无人连接的时间，用于超时清理。
        std::unordered_map<std::string,std::chrono::steady_clock::time_point>all_disconnected_since_;
    };
}
