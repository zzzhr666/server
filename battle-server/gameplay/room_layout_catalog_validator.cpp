#include "room_layout_catalog_validator.hpp"

#include <unordered_set>

std::vector<battle::RoomLayoutCatalogIssue> battle::validate_room_layout_catalog(
    const RoomLayoutCatalog& catalog) {
    std::unordered_set<std::string> seen_layout_ids;
    std::unordered_set<std::string> reported_duplicate_ids;
    std::vector<RoomLayoutCatalogIssue> issues;

    for (std::size_t layout_index = 0; layout_index < catalog.layouts.size(); ++layout_index) {
        const auto& layout = catalog.layouts[layout_index];
        if (seen_layout_ids.insert(layout.layout_id).second ||
            !reported_duplicate_ids.insert(layout.layout_id).second) {
            continue;
        }
        issues.emplace_back(RoomLayoutCatalogIssue{
            .kind = RoomLayoutCatalogIssueKind::DuplicateLayoutID,
            .layout_id = layout.layout_id,
            .layout_index = layout_index,
        });
    }

    return issues;
}
