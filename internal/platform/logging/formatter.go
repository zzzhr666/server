package logging

type Formatter interface {
	Format(record Record) string
}
