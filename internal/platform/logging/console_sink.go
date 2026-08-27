package logging

import "fmt"

type ConsoleSink struct{}

// Log 将格式化后的日志消息写入标准输出。
func (c *ConsoleSink) Log(message string) {
	fmt.Println(message)
}
