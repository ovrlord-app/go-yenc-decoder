package yenc

func NewOption[T any, O ~func(*T)](options []O, defaults ...O) *T {
	return ApplyOption(nil, options, defaults...)
}

func ApplyOption[T any, O ~func(*T)](opts *T, options []O, defaults ...O) *T {
	if opts == nil {
		opts = new(T)
	}
	for _, option := range defaults {
		option(opts)
	}
	for _, option := range options {
		option(opts)
	}
	return opts
}
