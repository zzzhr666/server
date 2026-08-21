cmake_minimum_required(VERSION 3.20)
project(battle_server)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

find_package(Protobuf REQUIRED)
find_package(gRPC CONFIG REQUIRED)
find_package(GTest REQUIRED)
find_package(spdlog CONFIG REQUIRED)
find_package(prometheus-cpp CONFIG REQUIRED)

enable_testing()

add_library(battle_proto
        generated/proto/battle/v1/battle.pb.cc
        generated/proto/battle/v1/battle.grpc.pb.cc
        generated/proto/battle/v1/session.pb.cc
        generated/proto/rcenter/v1/rcenter.pb.cc
        generated/proto/rcenter/v1/rcenter.grpc.pb.cc
)

add_library(battle_platform
        platform/command_line.cpp
        platform/config.cpp
        platform/logging.cpp
        platform/stop_signal.cpp
)

add_library(battle_metrics
        platform/metrics_server.cpp
        platform/metrics.cpp
)

add_library(battle_control
        control/control_handler.cpp
        control/grpc_server.cpp
)
add_library(battle_ecs
        ecs/world.cpp
        ecs/system/system_scheduler.cpp
        ecs/system/move_resolve_system.cpp
        ecs/system/move_system.cpp
        ecs/system/monster_ai_system.cpp
        ecs/system/attack_resolve_system.cpp
        ecs/system/hit_resolve_system.cpp
        ecs/system/damage_system.cpp
        ecs/system/death_system.cpp
        ecs/system/dash_system.cpp
        ecs/system/dash_resolve_system.cpp
        ecs/system/blessing_helpers.cpp
        ecs/system/damage_modify_system.cpp
        ecs/system/blessing_trigger_system.cpp
        ecs/system/status_effect_system.cpp
        ecs/system/pre_damage_blessing_system.cpp
        ecs/system/projectile_spawn_system.cpp
        ecs/system/projectile_range_system.cpp
        ecs/system/projectile_hit_system.cpp
        ecs/spatial_index.cpp
        ecs/system/trap_system.cpp
)
add_library(battle_game
        game/room.cpp
        game/game_manager.cpp
)
add_library(battle_gameplay
        gameplay/spawn_planner.cpp
        gameplay/monster_planner.cpp
        gameplay/monster_kind_codec.cpp
        gameplay/hero.cpp
        gameplay/growth.cpp
        gameplay/room_graph.cpp
        gameplay/room_graph_validator.cpp
        gameplay/room_layout_validator.cpp
        gameplay/room_layout_catalog.cpp
        gameplay/room_layout_catalog_validator.cpp
        gameplay/room_graph_presets.cpp
        gameplay/room_flow.cpp
        gameplay/room_encounter_validator.cpp
        gameplay/room_encounter_planner.cpp
        gameplay/room_runtime.cpp
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
        application.cpp
        main.cpp)

target_include_directories(battle_platform PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_link_libraries(battle_platform PUBLIC
		spdlog::spdlog)

target_include_directories(battle_metrics PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_link_libraries(battle_metrics PUBLIC
        prometheus-cpp::core
        prometheus-cpp::pull)

target_compile_definitions(battle_platform PUBLIC
        SPDLOG_ACTIVE_LEVEL=SPDLOG_LEVEL_TRACE)

target_include_directories(battle_game PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR})

target_link_libraries(battle_game PUBLIC
        battle_metrics)

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
        battle_runtime
        battle_proto
        spdlog::spdlog
)

target_link_libraries(battle_gameplay PUBLIC
        battle_ecs
        spdlog::spdlog
)

target_link_libraries(battle_session PUBLIC
        battle_game
        spdlog::spdlog
)

target_link_libraries(battle_runtime PUBLIC
        battle_game
        battle_session
        battle_gameplay
        battle_proto
        spdlog::spdlog
)

target_link_libraries(battle_net PUBLIC
        battle_runtime
        battle_session
        battle_proto
        spdlog::spdlog
)

target_link_libraries(battle_registry PUBLIC
        battle_game
        battle_platform
        battle_proto
        spdlog::spdlog
)

target_link_libraries(battle_server PRIVATE
        battle_metrics
        battle_platform
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
        gameplay/tests/growth_test.cpp
        gameplay/tests/room_graph_test.cpp
        gameplay/tests/spawn_planner_test.cpp
        gameplay/tests/hero_test.cpp
        runtime/tests/battle_instance_test.cpp
        runtime/tests/battle_runtime_test.cpp
)

add_executable(battle_platform_tests
        platform/tests/command_line_test.cpp
        platform/tests/metrics_test.cpp
)

target_link_libraries(battle_platform_tests PRIVATE
        battle_platform
        battle_metrics
        GTest::gtest_main
)

target_link_libraries(battle_runtime_tests PRIVATE
        battle_runtime
        battle_net
        GTest::gtest_main
)

include(GoogleTest)
gtest_discover_tests(battle_ecs_tests)
gtest_discover_tests(battle_runtime_tests)
gtest_discover_tests(battle_platform_tests)
