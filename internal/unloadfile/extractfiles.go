package unloadfile

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	e "github.com/jguillaumes/go-encoding/encodings"
	"github.com/jguillaumes/go-hexdump"
	xmit "github.com/jguillaumes/xmit_reader/internal/xmitfile"
)

var enc = e.NewEncoding()

func GenerateFiles(mMap MemberMap, unlFile *os.File, outdir string, extension string, xmf xmit.XmitFileParams, encoding string) (int, error) {
	numFiles := 0
	var err error
	for _, m := range mMap {
		mName := m.MemberName
		filepos := m.FilePtr
		fileName := filepath.Join(outdir, strings.Trim(mName, " ")+"."+strings.Trim(extension, " "))

		err = writeMember(unlFile, filepos, fileName, xmf, encoding)
		if err == nil {
			numFiles++
		} else {
			break
		}
	}
	return numFiles, err
}

func writeMember(f *os.File, fpos int64, outnam string, xmf xmit.XmitFileParams, encoding string) error {
	log.Debugf("Writing member data to %s\n", outnam)
	variableLength := (xmf.SourceRecfm[0] == 'V')
	lrecl := xmf.SourceLrecl
	recordBuffer := bytes.NewBuffer(make([]byte, 0, lrecl))

	_, err := f.Seek(fpos, io.SeekStart)
	if err != nil {
		return err
	}
	endMember := false

	memberFile, err := os.Create(outnam)

	if err != nil {
		log.Errorf("cannot create file %s: %v\n", outnam, err)
		return err
	}

	log.Infof("Writing file %s\n", outnam)
	defer memberFile.Close()

	for !endMember {
		blockheader := make([]byte, 8)
		nBlockRead, err := f.Read(blockheader)
		if err != nil {
			return err
		}
		if nBlockRead != 8 {
			return fmt.Errorf("expected to read 8 bytes, got %d", nBlockRead)
		}
		blocklen := binary.BigEndian.Uint16(blockheader[0:2])
		memberslen := blocklen - 8
		buffer := make([]byte, memberslen)
		nBlockRead, err = f.Read(buffer)
		if err != nil {
			return err
		}
		if nBlockRead != int(memberslen) {
			return fmt.Errorf("expected to read %d bytes, got %d", memberslen, nBlockRead)
		}
		b := bytes.NewBuffer(buffer)
		hdr := b.Next(12) // Block header
		blockFlag := hdr[0]
		if blockFlag != 0x00 {
			// Non member data block (notes or extended attributes), ignored
			log.Debugf("Non data bloc: %02x\n", blockFlag)
			continue
		} else if memberslen == 12 {
			// End of member marker
			endMember = true
			log.Debugf("EOB found")
			break
		} else {
			log.Debugf("Beginning of block")
		}
		log.Tracef("\n%s\n", hexdump.HexDump(hdr, encoding))

		recordBuffer.Reset()
		remainingRecord := int(lrecl) // Remaining record bytes to read
		for memberslen > 0 {
			blockSize := int(min(memberslen, 362))
			log.Tracef("Start of MDB")
			block := b.Next(blockSize)
			bb := bytes.NewBuffer(block)

			sliceLen := blockSize
			for nb := sliceLen; nb > 0; {
				if variableLength {
					log.Fatalln("Variable length records are not supported yet...")
					panic("Variable length records are not supported yet")
				} else {
					n, _ := recordBuffer.Write(bb.Next(remainingRecord))
					if n == 0 {
						// End of data reached
						break
					}
					remainingRecord -= n
					recordLine, _ := enc.DecodeBytes(recordBuffer.Bytes(), encoding)
					if remainingRecord == 0 {
						fmt.Fprintln(memberFile, recordLine)
						recordBuffer.Reset()
						remainingRecord = int(lrecl)
					}
					nb -= n
				}
			}

			memberslen -= uint16(blockSize)
		}
		memberFile.Close()
		endMember = true
	}

	return nil
}
