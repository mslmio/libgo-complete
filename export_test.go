package complete

// Options exposes options to the external tests, which can import predict.
func Options(c *Command, line string) []string {
	return c.options(line)
}
