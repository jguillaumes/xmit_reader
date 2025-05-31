package unloadfile

import (
	"encoding/json"
	"fmt"
	"io"
)

func ProcessUnloadFile(inFile io.Reader, targetDir string) (int, error) {
	var numbytes = 0

	data := make([]byte, Copyr1_size)
	n, err := inFile.Read(data)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if n < Copyr1_size {
		return 0, io.ErrUnexpectedEOF
	}
	numbytes += n
	c1, err := NewCopyr1(data)
	if err != nil {
		return 0, err
	}

	data = make([]byte, Copyr2_size)
	n, err = inFile.Read(data)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if n < Copyr2_size {
		return 0, io.ErrUnexpectedEOF
	}
	numbytes += n
	c2, err := NewCopyr2(data)
	if err != nil {
		return 0, err
	}
	marshalled, err := json.MarshalIndent(c1, "", "  ")
	fmt.Printf("COPYR1: %s\n", marshalled)
	marshalled, err = json.MarshalIndent(c2, "", "  ")
	fmt.Printf("COPYR2: %s\n", marshalled)
	return numbytes, nil
}
