#include "room_runtime.hpp"

#include "room_encounter_planner.hpp"
#include "room_graph.hpp"
#include "room_layout_catalog.hpp"

battle::RoomRuntime::RoomRuntime(const DungeonRoomGraph& dungeon_room_graph, const RoomLayoutCatalog& layout_catalog)
    : graph_(dungeon_room_graph), catalog_(layout_catalog), flow_(dungeon_room_graph.start_room_id) {}

bool battle::RoomRuntime::prepare_current_room() {
    if (state() != RoomFlowState::EnteringRoom) {
        return false;
    }
    const auto* current_room = graph_.find_room(current_room_id());
    if (current_room == nullptr) {
        return false;
    }

    const auto* current_layout = catalog_.find_layout(current_room->layout_id);
    if (current_layout == nullptr) {
        return false;
    }
    monster_configs_.clear();
    if (current_room->encounter.has_value()) {
        monster_configs_ = RoomEncounterPlanner::plan_encounter(
            current_room->encounter.value(), *current_layout);
    }
    obstacle_configs_.clear();
    for (const auto& obstacle : current_layout->obstacles) {
        obstacle_configs_.emplace_back(obstacle.center, obstacle.radius);
    }
    trap_configs_.clear();
    for (const auto& trap : current_layout->traps) {
        trap_configs_.emplace_back(trap.center, trap.radius, trap.kind);
    }
    return true;
}

bool battle::RoomRuntime::start_current_room() {
    auto* room = graph_.find_room(current_room_id());
    if (room == nullptr) {
        return false;
    }
    switch (room->kind) {
    case DungeonRoomKind::Reward:
        return flow_.transition_to(RoomFlowState::Rewarding);
    default:
        return flow_.transition_to(RoomFlowState::Fighting);
    }
}

bool battle::RoomRuntime::update_living_monster_count(std::size_t living_monster_count) {
    if (state() != RoomFlowState::Fighting) {
        return false;
    }
    if (living_monster_count > 0) {
        return false;
    }
    return flow_.transition_to(RoomFlowState::RoomCleared);
}

bool battle::RoomRuntime::begin_exit_selection() {
    return flow_.transition_to(RoomFlowState::ChoosingExit);
}

bool battle::RoomRuntime::select_exit(DungeonRoomID next_room_id) {
    return flow_.select_exit(next_room_id, graph_);
}

bool battle::RoomRuntime::complete_transition() {
    // 只有切房事务完成后才清理下一房间的待生成配置。
    if (!flow_.complete_transition()) {
        return false;
    }
    monster_configs_.clear();
    obstacle_configs_.clear();
    trap_configs_.clear();
    return true;
}

bool battle::RoomRuntime::begin_blessing_selection() {
    return flow_.transition_to(RoomFlowState::ChoosingBlessing);
}
