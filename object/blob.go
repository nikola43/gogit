package object

// WriteBlob writes a blob object and returns its hash.
func WriteBlob(root string, content []byte) (string, error) {
	return WriteObject(root, "blob", content)
}

// HashBlob returns the hash for a blob without writing it.
func HashBlob(content []byte) string {
	return HashObject("blob", content)
}

// ReadBlob reads a blob object and returns its content.
func ReadBlob(root, hash string) ([]byte, error) {
	_, content, err := ReadObject(root, hash)
	return content, err
}
