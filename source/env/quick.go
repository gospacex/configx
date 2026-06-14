package env

// Prefixed is the most common env constructor: it scopes a Source to
// variables starting with prefix+"_" and lower-cases keys.
//
//	env.Prefixed("APP")   // APP_SERVER_PORT → server.port
func Prefixed(prefix string) *Source { return New(WithPrefix(prefix)) }

// All exposes every environment variable as a (lower-cased) nested key.
// Use with caution — it can leak unrelated variables into your config tree.
func All() *Source { return New() }
