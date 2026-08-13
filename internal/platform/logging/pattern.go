package logging

type partKind uint8

const (
	partLiteral partKind = iota
	partYear
	partMonth
	partDay
	partHour
	partMinute
	partSecond
	partMillisecond
	partLevel
	partFile
	partFunction
	partLine
	partMessage
)

type formatPart struct {
	kind partKind
	text string
}

func parsePattern(pattern string) []formatPart {
	var parts []formatPart
	literalStart := 0
	for index := 0; index < len(pattern); {
		if pattern[index] != '%' {
			index++
			continue
		}
		if literalStart < index {
			parts = append(parts, formatPart{
				kind: partLiteral,
				text: pattern[literalStart:index],
			})
		}
		if index+1 >= len(pattern) {
			parts = append(parts, formatPart{
				kind: partLiteral,
				text: "%",
			})
			return parts
		}
		placeholder := pattern[index+1]
		switch placeholder {
		case '%':
			parts = append(parts, formatPart{
				kind: partLiteral,
				text: "%",
			})
		case 'Y':
			parts = append(parts, formatPart{
				kind: partYear,
			})
		case 'm':
			parts = append(parts, formatPart{
				kind: partMonth,
			})
		case 'd':
			parts = append(parts, formatPart{
				kind: partDay,
			})
		case 'H':
			parts = append(parts, formatPart{
				kind: partHour,
			})
		case 'M':
			parts = append(parts, formatPart{
				kind: partMinute,
			})
		case 'S':
			parts = append(parts, formatPart{
				kind: partSecond,
			})
		case 'e':
			parts = append(parts, formatPart{
				kind: partMillisecond,
			})
		case 'l':
			parts = append(parts, formatPart{
				kind: partLevel,
			})
		case 's':
			parts = append(parts, formatPart{
				kind: partFile,
			})
		case '!':
			parts = append(parts, formatPart{
				kind: partFunction,
			})
		case '#':
			parts = append(parts, formatPart{
				kind: partLine,
			})
		case 'v':
			parts = append(parts, formatPart{
				kind: partMessage,
			})
		default:
			parts = append(parts, formatPart{
				kind: partLiteral,
				text: pattern[index : index+2],
			})
		}
		index += 2
		literalStart = index
	}
	if literalStart < len(pattern) {
		parts = append(parts, formatPart{
			kind: partLiteral,
			text: pattern[literalStart:],
		})
	}
	return parts
}
