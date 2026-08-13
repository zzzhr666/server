#pragma once

#include <csignal>

namespace battle {
    /// @brief StopSignal 将 SIGINT 和 SIGTERM 转换为主线程可等待的停止通知。
    class StopSignal {
    public:
        StopSignal();
        ~StopSignal();

        StopSignal(const StopSignal&) = delete;
        StopSignal& operator=(const StopSignal&) = delete;

        /// @brief 阻塞当前线程，直到进程收到停止信号。
        void wait() const;

    private:
        struct sigaction previous_interrupt_ {};
        struct sigaction previous_terminate_ {};
    };
}
