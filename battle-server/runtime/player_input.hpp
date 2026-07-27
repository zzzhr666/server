#pragma once

namespace battle {
    struct PlayerInput {
        float move_x = 0.0f;
        float move_y = 0.0f;
        bool attack_requested = false;
        bool dash_requested = false;
    };
}
