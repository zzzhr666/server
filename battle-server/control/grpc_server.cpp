#include "grpc_server.hpp"

namespace {
    battle::CreateRoomRequest from_proto_request(const battle::v1::CreateRoomRequest& request) {
        battle::CreateRoomRequest out{
            .room_name = request.room_name(),
            .token = request.token()
        };
        out.player_ids.reserve(request.player_ids().size());
        for (const auto id : request.player_ids()) {
            out.player_ids.push_back(id);
        }
        return out;
    }

    battle::v1::CreateRoomStatus to_proto_status(battle::CreateRoomStatus status) {
        switch (status) {
        case battle::CreateRoomStatus::OK:
            return battle::v1::CREATE_ROOM_STATUS_OK;
        case battle::CreateRoomStatus::InvalidRequest:
            return battle::v1::CREATE_ROOM_STATUS_INVALID_REQUEST;
        case battle::CreateRoomStatus::AlreadyExists:
            return battle::v1::CREATE_ROOM_STATUS_ALREADY_EXISTS;
        case battle::CreateRoomStatus::InternalError:
            return battle::v1::CREATE_ROOM_STATUS_INTERNAL_ERROR;
        default:
            return battle::v1::CREATE_ROOM_STATUS_UNSPECIFIED;
        }
    }

    battle::JoinRoomRequest from_proto_request(const battle::v1::JoinRoomRequest& request) {
        return {
            .room_name = request.room_name(),
            .token = request.token(),
            .player_id = request.player_id()
        };
    }

    battle::EndRoomRequest from_proto_request(const battle::v1::EndRoomRequest& request) {
        return {
            .room_name = request.room_name(),
            .reason = request.reason(),
        };
    }

    battle::v1::JoinRoomStatus to_proto_status(battle::JoinRoomStatus status) {
        switch (status) {
        case battle::JoinRoomStatus::OK:
            return battle::v1::JOIN_ROOM_STATUS_OK;
        case battle::JoinRoomStatus::InvalidRequest:
            return battle::v1::JOIN_ROOM_STATUS_INVALID_REQUEST;
        case battle::JoinRoomStatus::RoomNotFound:
            return battle::v1::JOIN_ROOM_STATUS_ROOM_NOT_FOUND;
        case battle::JoinRoomStatus::InvalidToken:
            return battle::v1::JOIN_ROOM_STATUS_INVALID_TOKEN;
        case battle::JoinRoomStatus::PlayerNotAllowed:
            return battle::v1::JOIN_ROOM_STATUS_PLAYER_NOT_ALLOWED;
        case battle::JoinRoomStatus::AlreadyJoined:
            return battle::v1::JOIN_ROOM_STATUS_ALREADY_JOINED;
        case battle::JoinRoomStatus::InternalError:
            return battle::v1::JOIN_ROOM_STATUS_INTERNAL_ERROR;
        default:
            return battle::v1::JOIN_ROOM_STATUS_UNSPECIFIED;
        }
    }

    battle::v1::EndRoomStatus to_proto_status(battle::EndRoomStatus status) {
        switch (status) {
        case battle::EndRoomStatus::OK:
            return battle::v1::END_ROOM_STATUS_OK;
        case battle::EndRoomStatus::InvalidRequest:
            return battle::v1::END_ROOM_STATUS_INVALID_REQUEST;
        case battle::EndRoomStatus::RoomNotFound:
            return battle::v1::END_ROOM_STATUS_ROOM_NOT_FOUND;
        case battle::EndRoomStatus::InternalError:
            return battle::v1::END_ROOM_STATUS_INTERNAL_ERROR;
        default:
            return battle::v1::END_ROOM_STATUS_UNSPECIFIED;
        }
    }
}

battle::BattleControlServiceImpl::BattleControlServiceImpl(ControlHandler& handler)
    : control_handler_(handler) {}

grpc::Status battle::BattleControlServiceImpl::CreateRoom(grpc::ServerContext* context,
                                                          const v1::CreateRoomRequest* request,
                                                          v1::CreateRoomResponse* response) {
    (void)context;
    auto res = control_handler_.create_room(from_proto_request(*request));
    response->set_status(to_proto_status(res.status));
    response->set_message(res.message);
    return grpc::Status::OK;
}

grpc::Status battle::BattleControlServiceImpl::JoinRoom(grpc::ServerContext* context,
                                                        const v1::JoinRoomRequest* request,
                                                        v1::JoinRoomResponse* response) {
    (void)context;
    auto res = control_handler_.join_room(from_proto_request(*request));
    response->set_status(to_proto_status(res.status));
    response->set_message(res.message);
    return grpc::Status::OK;
}

grpc::Status battle::BattleControlServiceImpl::EndRoom(grpc::ServerContext* context,
                                                       const v1::EndRoomRequest* request,
                                                       v1::EndRoomResponse* response) {
    (void)context;
    auto res = control_handler_.end_room(from_proto_request(*request));
    response->set_status(to_proto_status(res.status));
    response->set_message(res.message);
    return grpc::Status::OK;
}
