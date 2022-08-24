package imagorpath

// StorageHasher define image key for storage
type StorageHasher interface {
	Hash(image string) string
}

// ResultStorageHasher define key for result storage
type ResultStorageHasher interface {
	HashResult(p Params) string
}
