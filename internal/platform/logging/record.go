package logging

import "time"

type Record struct {
	Time     time.Time
	Level    Level
	Message  string
	File     string
	Function string
	Line     int
}
