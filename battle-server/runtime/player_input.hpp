#pragma once

namespace battle {
    /// @brief 从 UDP 协议转换而来的玩家一帧输入。
    struct PlayerInput {
        float move_x = 0.0f;
        float move_y = 0.0f;
        bool attack_requested = false;
        bool dash_requested = false;
    };
}
