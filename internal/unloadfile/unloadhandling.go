package unloadfile

import (
	"bytes"
	"encoding/binary"
	"sort"

	"encoding/json"
	// "sort"

	// "encoding/json"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"os"

	"github.com/jguillaumes/go-ebcdic"
	"github.com/jguillaumes/go-hexdump"
	xmit "github.com/jguillaumes/xmit_reader/internal/xmitfile"
)

func ProcessUnloadFile(inFile os.File, targetDir string, typeExt string, xmf xmit.XmitFileParams) (int, error) {
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
	_ = c2
	if err != nil {
		return 0, err
	}

	if log.GetLevel() >= log.DebugLevel {
		marshalled, _ := json.MarshalIndent(c1, "", "  ")
		log.Debugln(string(marshalled))

		marshalled, _ = json.MarshalIndent(c2, "", "  ")
		log.Debugln(string(marshalled))
	}
	dirBlocks, err := readDirBlocks(inFile)
	if err != nil {
		return 0, err
	}

	if log.GetLevel() == log.TraceLevel {
		for i, d := range dirBlocks {
			log.Tracef("Directory block number %d\n", i)
			log.Tracef("\n%s\n", hexdump.HexDump(d[:], ebcdic.EBCDIC037))
		}
	}

	members, err := processDirBlocks(dirBlocks)
	if err != nil {
		return 0, err
	}

	if !c1.IsPdse() {
		// Jump over 12 "unknown" bytes in PDS unload
		dummyBuffer := make([]byte, 12)
		inFile.Read(dummyBuffer)
	}

	err = processDataRecords(inFile, members, c1.TracksPerCyl, c1, c2)
	if err != nil {
		return 0, err
	}

	_, err = GenerateFiles(members, &inFile, targetDir, typeExt, xmf)
	if err != nil {
		return 0, err
	}

	if log.GetLevel() == log.TraceLevel {

		keys := make([]uint32, 0, len(members))
		for k := range members {
			keys = append(keys, k)
		}

		sort.SliceStable(keys, func(i, j int) bool {
			//		return members[keys[i]].memberName < members[keys[j]].memberName
			var ttri uint32 = (uint32(members[keys[i]].Track) << 8) + uint32(members[keys[i]].Offset)
			var ttrj uint32 = (uint32(members[keys[j]].Track) << 8) + uint32(members[keys[j]].Offset)
			return ttri < ttrj
		})

		for k := range keys {
			m := members[keys[k]]
			log.Printf("Member %-8s(%06x): TT: 0x%04x, R: 0x%02x, Ptr: %016x\n", m.MemberName, keys[k], m.Track, m.Offset, m.FilePtr)
		}
	}

	if log.GetLevel() >= log.DebugLevel {
		marshalled, _ := json.MarshalIndent(c1, "", "  ")
		log.Debugf("COPYR1: %s\n", marshalled)
		marshalled, _ = json.MarshalIndent(c2, "", "  ")
		log.Debugf("COPYR2: %s\n", marshalled)
	}
	return numbytes, nil
}

func readDirBlocks(inFile os.File) ([]DirBlock, error) {
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
			return nil, fmt.Errorf("expected 8 bytes, read %d", n)
		}

		blockLen := binary.BigEndian.Uint16(headerBuffer[0:2]) - 8
		if blockLen == 12 {
			endDirBlocks = true
			_, err = inFile.Read(make([]byte, 12))
			if err != nil {
				return nil, err
			}
			break
		}
		numBlocks := blockLen / DirBlock_size
		for _ = range numBlocks {
			db := make([]byte, DirBlock_size)
			_, err := inFile.Read(db)
			if err == nil {
				dirBlocks = append(dirBlocks, DirBlock(db))
				log.Traceln(hexdump.HexDump(db, ebcdic.EBCDIC037))
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

func processDirBlocks(blocks []DirBlock) (MemberMap, error) {
	entries := make(map[uint32]MemberEntry, len(blocks))

	for _, b := range blocks {
		bbuff := bytes.NewBuffer(b[:])
		_ = bbuff.Next(12)
		lastEntry, _ := ebcdic.Decode(bbuff.Next(8), ebcdic.EBCDIC037)
		_ = bbuff.Next(2)
		endBlock := false
		for !endBlock {
			next8 := bbuff.Next(8)
			if bytes.Equal(next8, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}) {
				break
			}
			currEntry, _ := ebcdic.Decode(next8, ebcdic.EBCDIC037)
			tt := binary.BigEndian.Uint16(bbuff.Next(2))
			r, _ := bbuff.ReadByte()
			entry := MemberEntry{
				MemberName: currEntry,
				Track:      tt,
				Offset:     r,
			}
			ttr := uint32(tt)<<8 + uint32(r)
			entries[ttr] = entry
			if currEntry == lastEntry {
				break
			}
			// Skip user data if present
			c, _ := bbuff.ReadByte()
			userDataBytes := c & 0b00011111 * 2 // A mainframe halfword = 2 bytes
			_ = bbuff.Next(int(userDataBytes))
		}
	}
	return entries, nil
}

func processDataRecords(inFile os.File, members MemberMap, tpc uint16, cr1 *Copyr1, cr2 *Copyr2) error {

	// Read rest of records
	// The "header" portion is always 8 bytes
	rechead := make([]byte, 8)
	end_records := false
	for !end_records {
		currOffset, err := inFile.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		l, err := inFile.Read(rechead)
		if l != 8 || err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error reading record head, read %d byres: %v", l, err)
		}
		hbuff := bytes.NewBuffer(rechead)
		reclen := binary.BigEndian.Uint16(hbuff.Next(2))
		if reclen == 20 {
			// This is an end-of-member marker record, we skip it
			inFile.Read(make([]byte, 12))
			continue
		}
		_ = hbuff.Next(6)
		// Next byte will tell us if we are dealing with a member data record
		memberDataBuff := bytes.NewBuffer(make([]byte, reclen-8))
		_, err = inFile.Read(memberDataBuff.Bytes())
		if err != nil {
			return err
		}
		t, _ := memberDataBuff.ReadByte()
		if t != 0x00 {
			// Not a member data record, ignore
			continue
		} else {
			memberDataBuff.Next(3)                                        // Skip MBB
			cc := uint32(binary.BigEndian.Uint16(memberDataBuff.Next(2))) // Low 16 bits of cyl
			hh := binary.BigEndian.Uint16(memberDataBuff.Next(2))         // 12 hi bits of cyl + 4 bits of track/head
			cch := (hh & 0xFFF0) << 12                                    // Hi 12 bits of cyl (zero for non extended vols)
			ccl := cc + uint32(cch)                                       // Full cylinder number
			hht := 0x0F & hh                                              // Track/head number

			tt, err := findRelativeTrack(ccl, hht, cr1, cr2)
			if err != nil {
				log.Warnf("Cannot find relative track for cyl=%04x, head=%04x", cc, hh)
				continue
			}
			r, _ := memberDataBuff.ReadByte()
			ttr := tt<<8 + uint32(r)
			m, ok := members[ttr]
			if !ok {
				log.Warnf("Member with ttr %04x:%02x not found. len=%d, offset=%d (%04x%04x%02x)\n", ttr>>8, ttr&0xff, reclen, currOffset, cc, hh, r)
				log.Warnf("\n%s", hexdump.HexDump(memberDataBuff.Bytes()[0:64], ebcdic.EBCDIC037))
			} else {
				log.Debugf("Member with ttr %04x:%02x found (%s), len=%d, offset=%d\n", ttr>>8, ttr&0xff, m.MemberName, reclen, currOffset)
				log.Debugf("\n%s", hexdump.HexDump(memberDataBuff.Bytes()[0:64], ebcdic.EBCDIC037))
				m.FilePtr = currOffset
				members[ttr] = m
			}

		}
	}
	return nil
}

func findRelativeTrack(cc uint32, hh uint16, c1 *Copyr1, c2 *Copyr2) (uint32, error) {
	exts := &c2.Extensions

	var relTrack uint32 = 0
	for _, ext := range *exts {
		if ext.StartCylinder <= cc && cc <= ext.EndCylinder {
			relTrack += (cc-ext.StartCylinder)*uint32(c1.TracksPerCyl) + uint32(hh) - uint32(ext.StartTrack)
			break
		} else {
			relTrack += uint32(c1.TracksPerCyl)
		}
	}
	return uint32(relTrack), nil
}
