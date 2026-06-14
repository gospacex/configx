package loader

type ConfigLoader interface {
    Load(key string) ([]byte, error)
    Watch(key string, fn func([]byte))
    Close() error
}