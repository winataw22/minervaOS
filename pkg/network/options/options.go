package options

func flag(t bool) string {
	if t {
		return "1"
	}

	return "0"
}

// Option interface
type Option interface {
	apply(link string) error
}

// Set link options
func Set(link string, option ...Option) error {
	for _, opt := range option {
		if err := opt.apply(link); err != nil {
			return err
		}
	}

	return nil
}
