#include "udp_server.hpp"

#include <utility>
#include <arpa/inet.h>
#include <cstring>
#include <unistd.h>

#include "packet_codec.hpp"
#include "platform/metrics.hpp"
#include "runtime/battle_runtime.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"
#include "spdlog/spdlog.h"


battle::UdpServer::UdpServer(std::string listen_addr, SessionManager& session_manager, BattleMetrics& metrics)
    : listen_addr_(std::move(listen_addr)), session_manager_(session_manager), metrics_(metrics),
      battle_runtime_(nullptr), running_(false), fd_(-1),
      next_conv_(1) {}

bool battle::UdpServer::start() {
    // UDP socket 必须在地址解析和 bind 都成功后才公开运行状态，
    // 这样 stop() 与接收线程不会看到半初始化的文件描述符。
    fd_ = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd_ < 0) {
        SPDLOG_ERROR("create UDP socket failed");
        return false;
    }
    sockaddr_in addr{
        .sin_family = AF_INET,
    };
    if (!parse_listen_addr_(addr)) {
        SPDLOG_ERROR("parse UDP listen address failed addr={}", listen_addr_);
        close(fd_);
        fd_ = -1;
        return false;
    }


    if (bind(fd_, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) < 0) {
        SPDLOG_ERROR("bind UDP socket failed addr={}", listen_addr_);
        close(fd_);
        fd_ = -1;
        return false;
    }

    running_ = true;
    SPDLOG_INFO("UDP server started addr={}", listen_addr_);
    thread_ = std::thread([this]() {
        run_loop_();
    });
    return true;
}

void battle::UdpServer::stop() {
    running_ = false;
    if (fd_ >= 0) {
        close(fd_);
        fd_ = -1;
    }
    if (thread_.joinable()) {
        thread_.join();
    }
    SPDLOG_INFO("UDP server stopped");
}

void battle::UdpServer::set_runtime(BattleRuntime& battle_runtime) {
    battle_runtime_ = &battle_runtime;
}

void battle::UdpServer::run_loop_() {
    while (running_) {
        sockaddr_in remote_addr{};
        socklen_t len = sizeof(remote_addr);
        char buffer[4096];
        auto n = recvfrom(fd_, buffer, sizeof(buffer), 0, reinterpret_cast<sockaddr*>(&remote_addr), &len);
        if (n <= 0) {
            if (!running_) {
                break;
            }
            continue;
        }
        // 所有 UDP 入口先完成 protobuf 解码；后续处理器只接收结构化数据，
        // 避免把畸形字节包带入会话或战斗逻辑。
        auto packet = decode_client_packet(std::string_view{buffer, static_cast<std::size_t>(n)});
        if (!packet.has_value()) {
            metrics_.observe_udp_packet("received", false);
            SPDLOG_WARN("invalid UDP packet received");
            send_packet_(make_error("bad_packet", "decode client packet failed"), remote_addr, len);
            continue;
        }
        switch (packet->payload_case()) {
        case v1::ClientPacket::kHello: {
            metrics_.observe_udp_packet("received", true);
            handle_hello_(packet.value(), remote_addr, len);
            break;
        }

        case v1::ClientPacket::kInput: {
            metrics_.observe_udp_packet("received", true);
            handle_move_input_(packet.value(), remote_addr, len);
            break;
        }

        case v1::ClientPacket::kChooseBlessing: {
            metrics_.observe_udp_packet("received", true);
            handle_choose_blessing_(packet.value(), remote_addr, len);
            break;
        }
        case v1::ClientPacket::kHeartbeat: {
            metrics_.observe_udp_packet("received", true);
            handle_heartbeat_(packet.value(),remote_addr,len);
            break;
        }

        default: {
            metrics_.observe_udp_packet("received", false);
            send_packet_(make_error("unexpected_packet", "unexpected packet"), remote_addr, len);
        }
        }
    }
}

void battle::UdpServer::send_packet_(const v1::ServerPacket& packet, const sockaddr_in& remote_addr,
                                     socklen_t remote_addr_len) const {
    auto bytes = encode_server_packet(packet);
    if (sendto(fd_, bytes.data(), bytes.size(), 0,
               reinterpret_cast<const sockaddr*>(&remote_addr), remote_addr_len) < 0) {
        metrics_.observe_udp_packet("sent", false);
        SPDLOG_WARN("send UDP packet failed");
        return;
    }
    metrics_.observe_udp_packet("sent", true);
}

bool battle::UdpServer::parse_listen_addr_(sockaddr_in& out) const {
    const auto pos = listen_addr_.rfind(':');
    if (pos == std::string::npos) {
        return false;
    }

    const std::string ip = listen_addr_.substr(0, pos);
    const std::string port_str = listen_addr_.substr(pos + 1);

    int port = 0;
    try {
        port = std::stoi(port_str);
    } catch (...) {
        return false;
    }

    if (port <= 0 || port > 65535) {
        return false;
    }

    std::memset(&out, 0, sizeof(out));
    out.sin_family = AF_INET;
    out.sin_port = htons(static_cast<uint16_t>(port));

    if (inet_pton(AF_INET, ip.c_str(), &out.sin_addr) != 1) {
        return false;
    }

    return true;
}

void battle::UdpServer::handle_hello_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr,
                                      socklen_t remote_addr_len) {
    const auto& hello = packet.hello();
    if (hello.room_name().empty() || hello.token().empty() || hello.player_id() <= 0) {
        send_packet_(make_error("invalid_request", "invalid hello"), remote_addr, remote_addr_len);
        return;
    }
    // 每次 hello 分配新的 conversation。对于断线玩家，SessionManager 会原子地
    // 移除旧 conversation 索引并绑定新端点，从而让 NAT 变化后的客户端恢复同一局。
    auto conv = get_next_conv_();
    auto join_res = session_manager_.join({
        .room_name = hello.room_name(),
        .token = hello.token(),
        .player_id = hello.player_id(),
        .conv = conv,
        .endpoint = UdpEndpoint{remote_addr}
    });
    if (join_res.status == JoinSessionStatus::OK) {
        SPDLOG_INFO("UDP session joined room={} player={}", hello.room_name(), hello.player_id());
        send_packet_(make_server_hello(join_res.session->conv(), "session joined"), remote_addr, remote_addr_len);
        // 仅完整 roster 第一次入场时启动房间。重连返回 AlreadyJoined，不能重复
        // 创建 BattleInstance，也不能重新广播一次完整房间的启动流程。
        if (join_res.all_players_joined) {
            if (!battle_runtime_) {
                send_packet_(make_error("runtime_unavailable", "battle runtime is not attached"), remote_addr,
                             remote_addr_len);
                return;
            }
            battle_runtime_->start_room(std::string(join_res.session->room_name()));
        }
    } else if (join_res.status == JoinSessionStatus::AlreadyJoined && join_res.session) {
        SPDLOG_DEBUG("UDP session reconnected room={} player={}", hello.room_name(), hello.player_id());
        send_packet_(make_server_hello(join_res.session->conv(), "session already joined"), remote_addr,
                     remote_addr_len);
    } else {
        SPDLOG_WARN("UDP session join failed room={} player={} reason={}", hello.room_name(), hello.player_id(), join_res.message);
        send_packet_(make_error("join_failed", join_res.message), remote_addr, remote_addr_len);
    }
}

void battle::UdpServer::handle_move_input_(const v1::ClientPacket& packet,
                                           const sockaddr_in& remote_addr,
                                           socklen_t remote_addr_len) const {
    const auto& input = packet.input();
    if (input.room_name().empty() || input.player_id() <= 0) {
        send_packet_(make_error("invalid_request", "invalid move input"), remote_addr, remote_addr_len);
        return;
    }
    // 输入同时作为保活包。touch 会校验玩家、房间和源 UDP endpoint；因此不能
    // 伪造其他玩家 ID 来驱动其实体，也不会接受已经重绑前的旧端点。
    if (!session_manager_.touch(input.room_name(), input.player_id(),{remote_addr})) {
        send_packet_(make_error("invalid_session","session is not active"),remote_addr,remote_addr_len);
        return;
    }
    if (!battle_runtime_) {
        send_packet_(make_error("runtime_unavailable", "battle runtime is not attached"), remote_addr,
                     remote_addr_len);
        return;
    }
    if (!battle_runtime_->receive_input(input.room_name(), input.player_id(), PlayerInput{
                                            .move_x = input.x(),
                                            .move_y = input.y(),
                                            .attack_requested = input.attack_requested(),
                                            .dash_requested = input.dash_requested(),

                                        })) {
        send_packet_(make_error("internal_error", "unable to locate instance or entity"), remote_addr, remote_addr_len);
    }
}

void battle::UdpServer::handle_choose_blessing_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr,
                                                socklen_t remote_addr_len) const {
    const auto& choose_blessing = packet.choose_blessing();
    if (choose_blessing.room_name().empty() || choose_blessing.player_id() <= 0 || choose_blessing.option_id() < 0) {
        send_packet_(make_error("invalid_request", "invalid choose blessing"), remote_addr, remote_addr_len);
        return;
    }
    // 祝福选择和移动输入采用同一会话认证路径，防止断开或非本房间玩家影响结算阶段。
    if (!session_manager_.touch(choose_blessing.room_name(),choose_blessing.player_id(),{remote_addr})) {
        send_packet_(make_error("invalid_session", "session is not active"), remote_addr, remote_addr_len);
        return;
    }
    if (!battle_runtime_) {
        send_packet_(make_error("runtime_unavailable", "battle runtime is not attached"), remote_addr,
                     remote_addr_len);
        return;
    }
    if (!battle_runtime_->choose_blessing(choose_blessing.room_name(), choose_blessing.player_id(),
                                          choose_blessing.option_id())) {
        send_packet_(make_error("internal_error", "unable to choose blessing"), remote_addr, remote_addr_len);
    }
}

void battle::UdpServer::handle_heartbeat_(const v1::ClientPacket& packet, const sockaddr_in& remote_addr,
    socklen_t remote_addr_len) const {
    const auto& heartbeat = packet.heartbeat();
    if (heartbeat.room_name().empty() || heartbeat.player_id() <= 0) {
        send_packet_(make_error("invalid_request", "invalid heartbeat"), remote_addr, remote_addr_len);
        return;
    }
    // 无操作期间依靠心跳维持 Connected 状态；未通过端点校验的心跳不会延长会话寿命。
    if (!session_manager_.touch(heartbeat.room_name(), heartbeat.player_id(),{remote_addr})) {
        send_packet_(make_error("invalid_session","session is not active"), remote_addr, remote_addr_len);
    }
}

void battle::UdpServer::send_packet(const v1::ServerPacket& packet, const UdpEndpoint& endpoint) const {
    send_packet_(packet, endpoint.addr, sizeof(endpoint.addr));
}
