#pragma once

namespace battle {
    struct Config;

    enum class CommandLineResult {
        Ok,
        Help,
        Error,
    };

    /// @brief ParseCommandLine 将 battle-server 参数覆盖到默认配置。
    CommandLineResult ParseCommandLine(int argc, char* argv[], Config& config);
}
