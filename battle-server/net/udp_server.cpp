#include "udp_server.hpp"

#include <utility>
#include <arpa/inet.h>
#include <cstring>
#include <unistd.h>
#include <vector>

#include "packet_codec.hpp"
#include "runtime/battle_runtime.hpp"
#include "session/battle_session.hpp"
#include "session/session_manager.hpp"


battle::UdpServer::UdpServer(std::string listen_addr,RoomManager& room_manager, SessionManager& session_manager)
    : listen_addr_(std::move(listen_addr)), session_manager_(session_manager),
      battle_runtime_(room_manager,session_manager, [this](const v1::ServerPacket& p, const UdpEndpoint& ep) {
          send_packet_(p, ep.addr, sizeof(ep.addr));
      }), running_(false), fd_(-1),
      next_conv_(1) {}

bool battle::UdpServer::start() {
    fd_ = socket(AF_INET,SOCK_DGRAM, 0);
    if (fd_ < 0) {
        return false;
    }
    sockaddr_in addr{
        .sin_family = AF_INET,
    };
    if (!parse_listen_addr_(addr)) {
        close(fd_);
        fd_ = -1;
        return false;
    }


    if (bind(fd_, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) < 0) {
        close(fd_);
        fd_ = -1;
        return false;
    }

    running_ = true;
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
        auto packet = decode_client_packet(std::string_view{buffer, static_cast<std::size_t>(n)});
        if (!packet.has_value()) {
            send_packet_(make_error("bad_packet", "decode client packet failed"), remote_addr, len);
            continue;
        }
        if (packet->payload_case() != v1::ClientPacket::kHello) {
            send_packet_(make_error("unexpected_packet", "unexpected packet"), remote_addr, len);
            continue;
        }

        const auto& hello = packet->hello();
        if (hello.room_name().empty() || hello.token().empty() || hello.player_id() <= 0) {
            send_packet_(make_error("invalid_request", "invalid hello"), remote_addr, len);
            continue;
        }
        auto conv = get_next_conv_();
        auto join_res = session_manager_.join({
            .room_name = hello.room_name(),
            .token = hello.token(),
            .player_id = hello.player_id(),
            .conv = conv,
            .endpoint = UdpEndpoint{remote_addr}
        });
        if (join_res.status == JoinSessionStatus::OK) {
            send_packet_(make_server_hello(join_res.session->conv(), "session joined"), remote_addr, len);
            if (join_res.all_players_joined) {
                battle_runtime_.start_room(std::string(join_res.session->room_name()));
            }
        } else if (join_res.status == JoinSessionStatus::AlreadyJoined && join_res.session) {
            send_packet_(make_server_hello(join_res.session->conv(), "session already joined"), remote_addr, len);
        } else {
            send_packet_(make_error("join_failed", join_res.message), remote_addr, len);
        }
    }
}

void battle::UdpServer::send_packet_(const v1::ServerPacket& packet, const sockaddr_in& remote_addr,
                                     socklen_t remote_addr_len) {
    auto bytes = encode_server_packet(packet);
    if (sendto(fd_, bytes.data(), bytes.size(), 0,
               reinterpret_cast<const sockaddr*>(&remote_addr), remote_addr_len) < 0) {
        //todo： log send failure
    }
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
