#include "stop_signal.hpp"

#include <chrono>
#include <stdexcept>
#include <thread>

namespace {
    volatile std::sig_atomic_t stop_requested = 0;

    extern "C" void handle_stop_signal(int) {
        stop_requested = 1;
    }
}

battle::StopSignal::StopSignal() {
    stop_requested = 0;
    struct sigaction action {};
    action.sa_handler = handle_stop_signal;
    sigemptyset(&action.sa_mask);
    action.sa_flags = 0;
    if (sigaction(SIGINT, &action, &previous_interrupt_) != 0) {
        throw std::runtime_error("install SIGINT handler failed");
    }
    if (sigaction(SIGTERM, &action, &previous_terminate_) != 0) {
        sigaction(SIGINT, &previous_interrupt_, nullptr);
        throw std::runtime_error("install SIGTERM handler failed");
    }
}

battle::StopSignal::~StopSignal() {
    sigaction(SIGTERM, &previous_terminate_, nullptr);
    sigaction(SIGINT, &previous_interrupt_, nullptr);
}

void battle::StopSignal::wait() const {
    while (stop_requested == 0) {
        std::this_thread::sleep_for(std::chrono::milliseconds{100});
    }
}
