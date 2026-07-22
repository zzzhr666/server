cmake_minimum_required(VERSION 3.20)
project(battle_server)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

find_package(Protobuf CONFIG REQUIRED)
find_package(gRPC CONFIG REQUIRED)

add_library(battle_proto
        generated/proto/battle/v1/battle.pb.cc
        generated/proto/battle/v1/battle.grpc.pb.cc
)

add_library(battle_platform
        platform/config.cpp
)

add_library(battle_app INTERFACE)
add_library(battle_control
        control/control_handler.cpp
        control/grpc_server.cpp
        control/grpc_server.hpp
)
add_library(battle_ecs INTERFACE)
add_library(battle_game
        game/room.cpp
        game/game_manager.cpp
)
add_library(battle_net INTERFACE)

add_executable(battle_server
        main.cpp)

target_include_directories(battle_platform PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_game PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_control PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_proto PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR}/generated
)
target_link_libraries(battle_proto PUBLIC
        protobuf::libprotobuf
        gRPC::grpc++
)

target_link_libraries(battle_control PUBLIC
        battle_game
        battle_proto
)

target_link_libraries(battle_server PRIVATE
        battle_platform
        battle_app
        battle_control
        battle_ecs
        battle_game
        battle_net)
