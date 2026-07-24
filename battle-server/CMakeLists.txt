cmake_minimum_required(VERSION 3.20)
project(battle_server)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

find_package(Protobuf CONFIG REQUIRED)
find_package(gRPC CONFIG REQUIRED)
find_package(GTest CONFIG REQUIRED)

enable_testing()

add_library(battle_proto
        generated/proto/battle/v1/battle.pb.cc
        generated/proto/battle/v1/battle.grpc.pb.cc
        generated/proto/battle/v1/session.pb.cc
        generated/proto/rcenter/v1/rcenter.pb.cc
        generated/proto/rcenter/v1/rcenter.grpc.pb.cc
)

add_library(battle_platform
        platform/config.cpp
)

add_library(battle_app INTERFACE)
add_library(battle_control
        control/control_handler.cpp
        control/grpc_server.cpp
)
add_library(battle_ecs
        ecs/world.cpp
        ecs/system/system_scheduler.cpp
        ecs/system/system_scheduler.hpp
        ecs/system/input_system.cpp
        ecs/system/move_system.cpp
)
add_library(battle_game
        game/room.cpp
        game/game_manager.cpp
)
add_library(battle_gameplay
        gameplay/spawn_planner.cpp
)
add_library(battle_session
        session/battle_session.cpp
        session/session_manager.cpp
)
add_library(battle_runtime
        runtime/battle_runtime.cpp
        runtime/battle_instance.cpp
)
add_library(battle_net
        net/packet_codec.cpp
        net/udp_server.cpp
)

add_library(battle_registry
        registry/rcenter_client.cpp
        registry/node_registrar.cpp
)

add_executable(battle_server
        main.cpp)

target_include_directories(battle_platform PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_game PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_gameplay PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_session PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_runtime PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_net PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_control PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_registry PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_include_directories(battle_ecs PUBLIC
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

target_link_libraries(battle_gameplay PUBLIC
        battle_ecs
)

target_link_libraries(battle_session PUBLIC
        battle_game
)

target_link_libraries(battle_runtime PUBLIC
        battle_game
        battle_session
        battle_gameplay
        battle_proto
)

target_link_libraries(battle_net PUBLIC
        battle_runtime
        battle_session
        battle_proto
)

target_link_libraries(battle_registry PUBLIC
        battle_game
        battle_platform
        battle_proto
)

target_link_libraries(battle_server PRIVATE
        battle_platform
        battle_app
        battle_control
        battle_ecs
        battle_game
        battle_gameplay
        battle_net
        battle_registry)

add_executable(battle_ecs_tests
        ecs/tests/world_test.cpp
)

target_link_libraries(battle_ecs_tests PRIVATE
        battle_ecs
        GTest::gtest_main
)

add_executable(battle_runtime_tests
        gameplay/tests/spawn_planner_test.cpp
        runtime/tests/battle_instance_test.cpp
)

target_link_libraries(battle_runtime_tests PRIVATE
        battle_runtime
        GTest::gtest_main
)

include(GoogleTest)
gtest_discover_tests(battle_ecs_tests)
gtest_discover_tests(battle_runtime_tests)
