package logging

type Formatter interface {
	// Format 将结构化日志记录转换为 Sink 接收的文本。
	Format(record Record) string
}
