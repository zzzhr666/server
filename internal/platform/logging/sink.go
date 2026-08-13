package logging

// Sink 接收 Logger 格式化后的日志消息。
type Sink interface {
	Log(message string)
}

// Flusher 表示支持显式刷新待写数据的 Sink。
type Flusher interface {
	Flush() error
}

// Closer 表示持有需要释放资源的 Sink。
type Closer interface {
	Close() error
}
