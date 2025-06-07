package unloadfile

import (
	"bytes"
	"encoding/binary"
	"fmt"

	xu "github.com/jguillaumes/xmit_reader/internal/xmitutils"
)

const Copyr1_size = 64

type Copyr1 struct {
	DsFlags      uint8  `json:"dsflags"`
	DsOrg        string `json:"dsorg"`
	DsBlkSize    uint16 `json:"dsblksize"`
	DsLrecl      uint16 `json:"dslrecl"`
	DsRecfm      string `json:"dsrecfm"`
	DvaClass     string `json:"dvaclass"`
	DvaUnit      string `json:"dvaunit"`
	MaxBlock     uint16 `json:"maxblock"`
	MaxTrack     uint16 `json:"maxtrack"`
	NumCyls      uint16 `json:"numcylinders"`
	TracksPerCyl uint16 `json:"trackspercyl"`
}

func (c *Copyr1) IsPdse() bool {
	return c.DsFlags&0x01 != 0
}

func NewCopyr1(raw []byte) (*Copyr1, error) {
	if len(raw) != Copyr1_size {
		return nil, fmt.Errorf("invalid Copyr1 record length: expected %d, got %d", Copyr1_size, len(raw))
	}
	recordData := bytes.NewBuffer(raw)
	_ = recordData.Next(8)                                    // Skip the first 8 bytes (header)
	dsFlags, _ := recordData.ReadByte()                       // Unloaded dataset flags
	_ = recordData.Next(3)                                    // Skip the next 3 bytes (eyecatcher)
	dsOrgBytes := binary.BigEndian.Uint16(recordData.Next(2)) // DSORG halfword
	blkSize := binary.BigEndian.Uint16(recordData.Next(2))    // Block size
	lrecl := binary.BigEndian.Uint16(recordData.Next(2))      // Logical record length
	recfmByte, _ := recordData.ReadByte()                     // RECFM byte
	_ = recordData.Next(5)                                    // Skip the next 5 bytes (irrelevant data)
	w0 := uint32(binary.BigEndian.Uint32(recordData.Next(4)))
	var dva_class string
	if w0&0b00000000_00000000_10000000_00000000 != 0 {
		dva_class = "magtape"
	} else if w0&0b00000000_00000000_01000000_00000000 != 0 {
		dva_class = "UR"
	} else if w0&0b00000000_00000000_00100000_00000000 != 0 {
		dva_class = "dasd"
	} else if w0&0b00000000_00000000_00010000_00000000 != 0 {
		dva_class = "display"
	} else {
		dva_class = "char reader"
	}

	dva_unit_b := w0 & 0xff
	var dva_unit string
	switch dva_unit_b {
	case 0x04:
		dva_unit = "9345"
	case 0x0E:
		dva_unit = "3380"
	case 0x0F:
		dva_unit = "3390"
	case 0x03:
		if dva_class == "magtape" {
			dva_unit = "3420"
		} else {
			dva_unit = "1442"
		}
	case 0x80:
		dva_unit = "3480"
	case 0x81:
		dva_unit = "3490"
	case 0x83:
		dva_unit = "3590"
	case 0x01:
		dva_unit = "2540"
	case 0x06:
		dva_unit = "3505"
	case 0x08:
		dva_unit = "1403"
	case 0x09:
		dva_unit = "3211"
	case 0x0B:
		dva_unit = "3203"
	case 0x0C:
		dva_unit = "3525"
	}

	max_block := binary.BigEndian.Uint32(recordData.Next(4))
	num_cyls := binary.BigEndian.Uint16(recordData.Next(2))
	tracks_per_cyl := binary.BigEndian.Uint16(recordData.Next(2))
	max_track := binary.BigEndian.Uint16(recordData.Next(2))
	_ = recordData.Next(6) // SKip remaining bytes

	_ = dsOrgBytes // This is not used in the current implementation, but could be used to derive the DSORG string.

	c := &Copyr1{
		DsFlags:      dsFlags,
		DsOrg:        "",
		DsBlkSize:    blkSize,
		DsLrecl:      lrecl,
		DsRecfm:      xu.RecfmByteToString(recfmByte),
		DvaClass:     dva_class,
		DvaUnit:      dva_unit,
		MaxBlock:     uint16(max_block),
		MaxTrack:     max_track,
		NumCyls:      num_cyls,
		TracksPerCyl: tracks_per_cyl,
	}

	return c, nil
}

const Copyr2_size = 284

type Copyr2 struct {
	DebTailS   []byte          `json:"debtail"`
	Extensions []ExtensionData `json:"extensions"`
}

func (c *Copyr2) DebTail() []byte { return c.DebTailS }

func NewCopyr2(raw []byte) (*Copyr2, error) {
	if len(raw) != Copyr2_size {
		return nil, fmt.Errorf("invalid Copyr2 record length: expected %d, got %d", Copyr2_size, len(raw))
	}
	recordData := bytes.NewBuffer(raw)
	extensions := make([]ExtensionData, 0, 16)

	_ = recordData.Next(8)         // Skip the first 8 bytes (header)
	debTail := recordData.Next(16) // Read the next 16 bytes as DebTail
	for i := 0; i < 16; i++ {
		_ = recordData.Next(5) // Skip DEBUCBAD and DEBDVMOD31
		hiTracks, _ := recordData.ReadByte()
		loStartCyl := binary.BigEndian.Uint16(recordData.Next(2))
		hiStartCylTrk := binary.BigEndian.Uint16(recordData.Next(2))
		loEndCyl := binary.BigEndian.Uint16(recordData.Next(2))
		hiEndCylTrk := binary.BigEndian.Uint16(recordData.Next(2))
		loTracks := binary.BigEndian.Uint16(recordData.Next(2))
		tracks := uint32(loTracks) + (uint32(hiTracks) << 16)
		startCyl := uint32(loStartCyl) + uint32((hiStartCylTrk&0xFFF0)<<12)
		startTrack := uint8(hiStartCylTrk & 0x0F)
		endCyl := uint32(loEndCyl) + uint32((hiEndCylTrk&0xFFF0)<<12)
		endTrack := uint8(hiEndCylTrk & 0x0F)
		extension := ExtensionData{
			NumTracks:     tracks,
			StartCylinder: startCyl,
			StartTrack:    startTrack,
			EndCylinder:   endCyl,
			EndTrack:      endTrack,
		}
		extensions = append(extensions, extension)
	}

	c := Copyr2{
		DebTailS:   debTail,
		Extensions: extensions,
	}

	return &c, nil
}

const DirBlock_size = 276

type DirBlock [DirBlock_size]byte

type MemberEntry struct {
	MemberName string
	Track      uint16
	Offset     uint8
	FilePtr    int64
}

type MemberMap map[uint32]MemberEntry

type ExtensionData struct {
	NumTracks     uint32 `json:"numtracks"`
	StartCylinder uint32 `json:"startcylinder"`
	StartTrack    uint8  `json:"starttrack"`
	EndCylinder   uint32 `json:"endcylinder"`
	EndTrack      uint8  `json:"endtrack"`
}
