package unloadfile

import (
	"bytes"
	"encoding/binary"
	"fmt"

	xu "github.com/jguillaumes/xmit_reader/internal/xmitutils"
)

type Copyr1 interface {
	DsFlags() uint8
	DsOrg() string
	DsBlksize() uint16
	DsLrecl() uint16
	DsRecfm() string
	RawDevInfo() []byte
}

const Copyr1_size = 64

type copyr1Impl struct {
	Copyr1      `json:"-"`
	DsFlagsB    uint8  `json:"dsflags"`
	DsOrgS      string `json:"dsorg"`
	DsBlkSizeS  uint16 `json:"dsblksize"`
	DsLreclS    uint16 `json:"dslrecl"`
	DsRecfmS    string `json:"dsrecfm"`
	RawDevInfoB []byte `json:"rawdevinfo"`
}

func (c *copyr1Impl) DsFlags() uint8     { return c.DsFlagsB }
func (c *copyr1Impl) DsOrg() string      { return c.DsOrgS }
func (c *copyr1Impl) DsBlksize() uint16  { return c.DsBlkSizeS }
func (c *copyr1Impl) DsLrecl() uint16    { return c.DsLreclS }
func (c *copyr1Impl) DsRecfm() string    { return c.DsRecfmS }
func (c *copyr1Impl) RawDevInfo() []byte { return c.RawDevInfoB }

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
	rawDevInfo := recordData.Next(20)

	_ = dsOrgBytes // This is not used in the current implementation, but could be used to derive the DSORG string.

	c := &copyr1Impl{
		DsFlagsB:    dsFlags,
		DsOrgS:      "",
		DsBlkSizeS:  blkSize,
		DsLreclS:    lrecl,
		DsRecfmS:    xu.RecfmByteToString(recfmByte),
		RawDevInfoB: rawDevInfo,
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
