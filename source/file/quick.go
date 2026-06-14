package file

// YAML returns a file source for the given path, forcing the YAML codec.
// Useful when the file does not carry a .yaml/.yml extension (e.g. piped
// through a Kubernetes ConfigMap mounted as "config").
func YAML(path string, opts ...Option) *Source {
	return New(path, append([]Option{WithFormat("yaml")}, opts...)...)
}

// JSON returns a file source forced to the JSON codec.
func JSON(path string, opts ...Option) *Source {
	return New(path, append([]Option{WithFormat("json")}, opts...)...)
}

// TOML returns a file source forced to the TOML codec.
func TOML(path string, opts ...Option) *Source {
	return New(path, append([]Option{WithFormat("toml")}, opts...)...)
}

// Optional returns a file source that loads to an empty map when the file
// is missing. Equivalent to New(path, WithOptional(true), ...).
func Optional(path string, opts ...Option) *Source {
	return New(path, append([]Option{WithOptional(true)}, opts...)...)
}
