package logging

import "fmt"

type ConsoleSink struct{}

func (c *ConsoleSink) Log(message string) {
	fmt.Println(message)
}
