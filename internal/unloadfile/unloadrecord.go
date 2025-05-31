package unloadfile

import (
	"encoding/binary"
	"fmt"
)

type Copyr1 interface {
	DsFlags() uint8
	DsOrg() string
	DsBlksize() uint16
	DsLrecl() uint16
	DsRecfm() string
	RawDevInfo() []byte
}

const Copyr1_size = 56

type copyr1Impl struct {
	Copyr1
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
		return nil, fmt.Errorf("Invalid Copyr1 record length: expected %d, got %d", Copyr1_size, len(raw))
	}
	dsFlags := raw[0]
	dsOrgBytes := binary.BigEndian.Uint16(raw[4:6])
	blkSize := binary.BigEndian.Uint16(raw[6:8])
	lrecl := binary.BigEndian.Uint16(raw[8:10])
	recfmBytes := raw[10]
	rawDevInfo := raw[16:36]

	_ = dsOrgBytes // This is not used in the current implementation, but could be used to derive the DSORG string.
	_ = recfmBytes // This is not used in the current implementation, but could be used to derive the RECFM string.

	c := &copyr1Impl{
		DsFlagsB:    dsFlags,
		DsOrgS:      "",
		DsBlkSizeS:  blkSize,
		DsLreclS:    lrecl,
		DsRecfmS:    "",
		RawDevInfoB: rawDevInfo,
	}

	return c, nil
}

type Copyr2 interface {
	DebTail() []byte
	DebExt() [][]byte
}

const Copyr2_size = 276

type copyr2Impl struct {
	Copyr2
	DebTailS []byte   `json:"debtail"`
	DebExtS  [][]byte `json:"debext"`
}

func (c *copyr2Impl) DebTail() []byte  { return c.DebTailS }
func (c *copyr2Impl) DebExt() [][]byte { return c.DebExtS }

func NewCopyr2(raw []byte) (Copyr2, error) {
	if len(raw) != Copyr2_size {
		return nil, fmt.Errorf("Invalid Copyr2 record length: expected %d, got %d", Copyr2_size, len(raw))
	}

	debTail := raw[0:16]
	debExt := make([][]byte, 0, 16)
	for i := 0; i < 16; i++ {
		start := 16 + 16*i
		end := start + 16
		oneDebTail := raw[start:end]
		debExt = append(debExt, oneDebTail)
	}

	c := &copyr2Impl{
		DebTailS: debTail,
		DebExtS:  debExt,
	}

	return c, nil
}
