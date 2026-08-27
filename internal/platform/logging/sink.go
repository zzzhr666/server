package logging

// Sink 接收 Logger 格式化后的日志消息。
type Sink interface {
	// Log 接收一条已格式化日志消息。
	Log(message string)
}

// Flusher 表示支持显式刷新待写数据的 Sink。
type Flusher interface {
	// Flush 将待写日志同步到底层输出。
	Flush() error
}

// Closer 表示持有需要释放资源的 Sink。
type Closer interface {
	// Close 释放 Sink 持有的资源。
	Close() error
}
