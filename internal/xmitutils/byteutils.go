package xmitutils

// getVariableLenghInt reads a variable-length integer from the provided byte slice.
// The number of bytes to read is specified by numbytes.
// It returns the integer value and an error if the data is insufficient.
// If the data is less than numbytes, it returns io.ErrShortBuffer.
func GetVariableLengthInt(numbytes int, data []byte) int {

	var value int
	for i := range int(numbytes) {
		value = (value << 8) | int(data[i])
	}
	return value
}
