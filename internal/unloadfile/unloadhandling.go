package unloadfile

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/jguillaumes/go-ebcdic"
	hexdump "github.com/jguillaumes/go-hexdump"
)

func ProcessUnloadFile(inFile io.Reader, targetDir string) (int, error) {
	var numbytes = 0

	//+
	// Read COPYR1 record
	//+
	copyr1Buffer := bytes.NewBuffer(make([]byte, Copyr1_size))
	n, err := inFile.Read(copyr1Buffer.Bytes())
	if n != Copyr1_size || err != nil {
		if err != nil {
			return 0, fmt.Errorf("failed to read COPYR1 record: %w", err)
		} else {
			return 0, fmt.Errorf("expected %d bytes for COPYR1 record, got %d bytes", Copyr1_size, n)
		}
	}
	c1, err := NewCopyr1(copyr1Buffer.Bytes())
	if err != nil {
		return 0, err
	}

	//+
	// Read COPYR2 record
	//+
	copyr2Buffer := bytes.NewBuffer(make([]byte, Copyr2_size))
	n, err = inFile.Read(copyr2Buffer.Bytes())
	if n != Copyr2_size || err != nil {
		if err != nil {
			return 0, fmt.Errorf("failed to read COPYR2 record: %w", err)
		} else {
			return 0, fmt.Errorf("expected %d bytes for COPYR2 record, got %d bytes", Copyr2_size, n)
		}
	}
	c2, err := NewCopyr2(copyr2Buffer.Bytes())
	if err != nil {
		return 0, err
	}

	dirBlocks, err := readDirBlocks(inFile)
	if err != nil {
		return 0, err
	}

	for i, d := range dirBlocks {
		log.Printf("Directory block number %d\n", i)
		log.Printf("\n%s\n", hexdump.HexDump(d[:], ebcdic.EBCDIC037))
	}

	// Read rest of records
	// The "header" portion is always 8 bytes
	rechead := make([]byte, 8)
	end_records := false
	for !end_records {
		l, err := inFile.Read(rechead)
		if l != 8 || err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error reading record head, read %d byres: %v", l, err)
		}
		hbuff := bytes.NewBuffer(rechead)
		reclen := binary.BigEndian.Uint16(hbuff.Next(2))
		_ = hbuff.Next(6)
		// Next byte will tell us if we are dealing with a member data record
		memberDataBuff := bytes.NewBuffer(make([]byte, reclen-8))
		l, err = inFile.Read(memberDataBuff.Bytes())
		log.Printf("\nPossible data record\n%s\n", hexdump.HexDump(memberDataBuff.Bytes(), ebcdic.EBCDIC037))
		t, _ := memberDataBuff.ReadByte()
		if t != 0x00 {
			// Not a member data record, ignore
			continue
		}

	}

	marshalled, err := json.MarshalIndent(c1, "", "  ")
	log.Printf("COPYR1: %s\n", marshalled)
	marshalled, err = json.MarshalIndent(c2, "", "  ")
	log.Printf("COPYR2: %s\n", marshalled)
	return numbytes, nil
}

func readDirBlocks(inFile io.Reader) ([]DirBlock, error) {
	dirBlocks := make([]DirBlock, 0)
	headerBuffer := make([]byte, 8)

	endDirBlocks := false

	for !endDirBlocks {
		n, err := inFile.Read(headerBuffer)
		if err != nil && err != io.EOF {
			return nil, err
		} else if err == io.EOF {
			endDirBlocks = true
		} else if n != 8 {
			return nil, fmt.Errorf("Expected 8 bytes, read %d", n)
		}

		blockLen := binary.BigEndian.Uint16(headerBuffer[0:2]) - 8
		if blockLen == 12 {
			endDirBlocks = true
			_, _ = inFile.Read(make([]byte, 12))
			break
		}
		numBlocks := blockLen / DirBlock_size
		for _ = range numBlocks {
			db := make([]byte, DirBlock_size)
			_, err := inFile.Read(db)
			if err == nil {
				dirBlocks = append(dirBlocks, DirBlock(db))
				// log.Printf("\n%s\n", hexdump.HexDump(db, ebcdic.EBCDIC037))
			} else if err == io.EOF {
				endDirBlocks = true
			} else {
				return nil, err
			}
		}
		if blockLen%DirBlock_size != 0 {
			endDirBlocks = true
		}

	}
	return dirBlocks, nil
}
