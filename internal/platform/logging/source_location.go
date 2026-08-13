package logging

import "runtime"

type SourceLocation struct {
	File     string
	Function string
	Line     int
}

func captureSourceLocation() SourceLocation {
	pc, file, line, ok := runtime.Caller(3)
	if !ok {
		return SourceLocation{}
	}

	function := ""
	if fn := runtime.FuncForPC(pc); fn != nil {
		function = fn.Name()
	}

	return SourceLocation{
		File:     file,
		Function: function,
		Line:     line,
	}
}
