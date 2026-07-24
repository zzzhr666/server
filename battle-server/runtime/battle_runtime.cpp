#include "battle_runtime.hpp"

#include <utility>
#include <vector>
#include <cstdint>
#include <chrono>

#include "battle_instance.hpp"
#include "game/game_manager.hpp"
#include "net/packet_codec.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"

namespace {
    battle::v1::ServerPacket make_snapshot(const std::string& room_name, const battle::ecs::WorldSnapshot& snapshot) {
        battle::v1::ServerPacket packet;
        auto send_pkg = packet.mutable_snapshot();
        send_pkg->set_room_name(room_name);
        for (auto& entity : snapshot.entities) {
            auto entity_snapshot = send_pkg->add_entities();
            entity_snapshot->set_entity(entity.entity);
            entity_snapshot->set_x_position(entity.x_position);
            entity_snapshot->set_y_position(entity.y_position);
            entity_snapshot->set_x_direction(entity.x_direction);
            entity_snapshot->set_y_direction(entity.y_direction);
            entity_snapshot->set_current_health(entity.current_health);
            entity_snapshot->set_max_health(entity.max_health);
        }
        return packet;
    }
}

battle::BattleRuntime::BattleRuntime(RoomManager& room_manager, SessionManager& session_manager,
                                     SendPacketCallback callback)
    : room_manager_(room_manager), session_manager_(session_manager),
      send_packet_(std::move(callback)), running_(false) {}

battle::BattleRuntime::~BattleRuntime() {
    stop();
}

void battle::BattleRuntime::start_room(const std::string& room_name) {
    {
        std::lock_guard<std::mutex> lock(mutex_);
        if (starting_rooms_.contains(room_name) || instances_.contains(room_name)) {
            return;
        }
        starting_rooms_.insert(room_name);
    }

    auto sessions = session_manager_.sessions_in_room(room_name);

    std::vector<std::int64_t> player_ids;
    player_ids.reserve(sessions.size());
    for (const auto& session : sessions) {
        player_ids.push_back(session->player_id());
    }

    auto instance = std::make_unique<BattleInstance>(BattleInstanceConfig{
        .room_name = room_name,
        .player_ids = player_ids,
    });

    auto game_start_packet = make_game_start(room_name, player_ids);

    {
        std::lock_guard<std::mutex> lock(mutex_);
        starting_rooms_.erase(room_name);
        instances_.emplace(room_name, std::move(instance));
    }

    for (const auto& session : sessions) {
        send_packet_(game_start_packet, session->endpoint());
    }
}

void battle::BattleRuntime::tick(ecs::DeltaTime delta_time) {
    std::vector<v1::ServerPacket> packets;
    std::vector<UdpEndpoint> endpoints;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        for (auto& [room_name, instance] : instances_) {
            instance->tick(delta_time);
            auto snapshot = instance->snapshot();
            const auto packet = make_snapshot(room_name, snapshot);
            auto sessions = session_manager_.sessions_in_room(room_name);
            for (auto& session : sessions) {
                packets.emplace_back(packet);
                endpoints.push_back(session->endpoint());
            }
        }
    }
    for (std::size_t i = 0; i < packets.size(); i++) {
        send_packet_(packets[i], endpoints[i]);
    }
}

bool battle::BattleRuntime::receive_input(const std::string& room_name, std::int64_t player_id, float x, float y) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = instances_.find(room_name);
    if (it == instances_.end()) {
        return false;
    }
    return it->second->receive_input(player_id, x, y);
}

void battle::BattleRuntime::start() {
    bool expected = false;
    if (!running_.compare_exchange_strong(expected, true)) {
        return;
    }
    tick_thread_ = std::thread([this]() {
        using clock = std::chrono::steady_clock;
        constexpr auto tick_interval = std::chrono::milliseconds(50);
        auto last_tick = clock::now();
        while (running_) {
            auto now = clock::now();
            const ecs::DeltaTime delta = now - last_tick;
            last_tick = now;
            tick(delta);

            std::this_thread::sleep_for(tick_interval);
        }
    });
}

void battle::BattleRuntime::stop() {
    running_ = false;
    if (tick_thread_.joinable()) {
        tick_thread_.join();
    }
}

battle::EndRoomResult battle::BattleRuntime::end_room(const std::string& room_name, const std::string& reason) {
    v1::ServerPacket packet;
    std::vector<UdpEndpoint> endpoints;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = instances_.find(room_name);
        if (it == instances_.end()) {
            return {
                .status = EndRoomStatus::RoomNotFound,
                .message = "unable to find instance",
            };
        }
        auto sessions = session_manager_.sessions_in_room(room_name);
        std::vector<std::int64_t> player_ids;
        player_ids.reserve(sessions.size());
        for (const auto& session : sessions) {
            player_ids.push_back(session->player_id());
            endpoints.push_back(session->endpoint());
        }
        packet = make_game_over(room_name, player_ids, reason);
        instances_.erase(it);
    }
    for (auto& endpoint : endpoints) {
        send_packet_(packet, endpoint);
    }
    session_manager_.remove_room(room_name);
    room_manager_.close_room(room_name);
    return {
        .status = EndRoomStatus::OK,
        .message = "room ended",
    };
}
