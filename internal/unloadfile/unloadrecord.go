package unloadfile

import (
	"bytes"
	"encoding/binary"
	"fmt"

	xu "github.com/jguillaumes/xmit_reader/internal/xmitutils"
)

type Copyr1 interface {
	DsFlags() uint8
	IsPdse() bool
	DsOrg() string
	DsBlksize() uint16
	DsLrecl() uint16
	DsRecfm() string
	DvaClass() string
	DvaUnit() string
	MaxBlock() uint16
	NumCylinders() uint16
	TracksPerCyl() uint16
}

const Copyr1_size = 64

type copyr1Impl struct {
	Copyr1     `json:"-"`
	DsFlagsB   uint8  `json:"dsflags"`
	DsOrgS     string `json:"dsorg"`
	DsBlkSizeS uint16 `json:"dsblksize"`
	DsLreclS   uint16 `json:"dslrecl"`
	DsRecfmS   string `json:"dsrecfm"`
	DvaClassS  string `json:"dvaclass"`
	DvaUnitS   string `json:"dvaunit"`
	MaxBlockH  uint16 `json:"maxblock"`
	NumCylsH   uint16 `json:"numcylinders"`
	TperCyl    uint16 `json:"trackspercyl"`
}

func (c *copyr1Impl) DsFlags() uint8       { return c.DsFlagsB }
func (c *copyr1Impl) IsPdse() bool         { return c.DsFlagsB&0x01 != 0 }
func (c *copyr1Impl) DsOrg() string        { return c.DsOrgS }
func (c *copyr1Impl) DsBlksize() uint16    { return c.DsBlkSizeS }
func (c *copyr1Impl) DsLrecl() uint16      { return c.DsLreclS }
func (c *copyr1Impl) DsRecfm() string      { return c.DsRecfmS }
func (c *copyr1Impl) DvaClass() string     { return c.DvaClassS }
func (c *copyr1Impl) DvaUnit() string      { return c.DvaUnitS }
func (c *copyr1Impl) MaxBlock() uint16     { return c.MaxBlockH }
func (c *copyr1Impl) NumCylinders() uint16 { return c.NumCylsH }
func (c *copyr1Impl) TracksPerCyl() uint16 { return c.TperCyl }

func NewCopyr1(raw []byte) (Copyr1, error) {
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
	_ = recordData.Next(3) // SKip remaining words

	_ = dsOrgBytes // This is not used in the current implementation, but could be used to derive the DSORG string.

	c := &copyr1Impl{
		DsFlagsB:   dsFlags,
		DsOrgS:     "",
		DsBlkSizeS: blkSize,
		DsLreclS:   lrecl,
		DsRecfmS:   xu.RecfmByteToString(recfmByte),
		DvaClassS:  dva_class,
		DvaUnitS:   dva_unit,
		MaxBlockH:  uint16(max_block),
		NumCylsH:   num_cyls,
		TperCyl:    tracks_per_cyl,
	}

	return c, nil
}

type Copyr2 interface {
	DebTail() []byte
	DebExt() [][]byte
}

const Copyr2_size = 284

type copyr2Impl struct {
	Copyr2   `json:"-"`
	DebTailS []byte   `json:"debtail"`
	DebExtS  [][]byte `json:"debext"`
}

func (c *copyr2Impl) DebTail() []byte  { return c.DebTailS }
func (c *copyr2Impl) DebExt() [][]byte { return c.DebExtS }

func NewCopyr2(raw []byte) (Copyr2, error) {
	if len(raw) != Copyr2_size {
		return nil, fmt.Errorf("invalid Copyr2 record length: expected %d, got %d", Copyr2_size, len(raw))
	}
	recordData := bytes.NewBuffer(raw)

	_ = recordData.Next(8)         // Skip the first 8 bytes (header)
	debTail := recordData.Next(16) // Read the next 16 bytes as DebTail
	debExt := make([][]byte, 0, 16)
	for i := 0; i < 16; i++ {
		debExt = append(debExt, recordData.Next(16)) // Read the next 16 bytes for each DebExt entry
	}

	c := &copyr2Impl{
		DebTailS: debTail,
		DebExtS:  debExt,
	}

	return c, nil
}

const DirBlock_size = 276

type DirBlock [DirBlock_size]byte

type MemberEntry struct {
	memberName string
	track      uint16
	offset     uint8
	filePtr    int64
}

type MemberMap map[uint32]MemberEntry
