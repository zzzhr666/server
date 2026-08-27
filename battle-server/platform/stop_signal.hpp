#pragma once

#include <csignal>

namespace battle {
    /// @brief StopSignal 将 SIGINT 和 SIGTERM 转换为主线程可等待的停止通知。
    class StopSignal {
    public:
        /// @brief 安装进程停止信号处理器。
        StopSignal();
        /// @brief 恢复 StopSignal 占用的进程信号资源。
        ~StopSignal();

        /// @brief StopSignal 持有进程级信号状态，不允许复制。
        StopSignal(const StopSignal&) = delete;
        /// @brief StopSignal 持有进程级信号状态，不允许复制赋值。
        StopSignal& operator=(const StopSignal&) = delete;

        /// @brief 阻塞当前线程，直到进程收到停止信号。
        void wait() const;

    private:
        struct sigaction previous_interrupt_ {};
        struct sigaction previous_terminate_ {};
    };
}
