package cache

// Cache is an interface that wraps multiple key vale stores
type Cache interface {
	Load(key string) ([]byte, error)
	LoadAll() (map[string][]byte, error)
	Store(map[string][]byte) error
	Close()
}
