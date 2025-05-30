package internal

import (
	"bytes"
	"encoding/binary"
)

type XmitTextUnitId uint16

const (
	XtuINMBLKSZ XmitTextUnitId = 0x0030 // Block size
	XtuINMCREAT XmitTextUnitId = 0x1022 // Creation date
	XtuINMDDNAM XmitTextUnitId = 0x0001 // DDNAME for the file
	XtuINMDIR   XmitTextUnitId = 0x000C // Number of directory blocks
	XtuINMDSNAM XmitTextUnitId = 0x0002 // Name of the file
	XtuINMDSORG XmitTextUnitId = 0x003C // File organization
	XtuINMEATTR XmitTextUnitId = 0x8028 // Extended attribute status
	XtuINMERRCD XmitTextUnitId = 0x1027 // RECEIVE command error code
	XtuINMEXPDT XmitTextUnitId = 0x0022 // Expiration date
	XtuINMFACK  XmitTextUnitId = 0x1026 // Originator requested notification
	XtuINMFFM   XmitTextUnitId = 0x102D // Filemode number
	XtuINMFNODE XmitTextUnitId = 0x1011 // Origin node name or node number
	XtuINMFTIME XmitTextUnitId = 0x1024 // Origin timestamp
	XtuINMFUID  XmitTextUnitId = 0x1012 // Origin user ID
	XtuINMFVERS XmitTextUnitId = 0x1023 // Origin version number of the data format
	XtuINMLCHG  XmitTextUnitId = 0x1021 // Date last changed
	XtuINMLRECL XmitTextUnitId = 0x0042 // Logical record length
	XtuINMLREF  XmitTextUnitId = 0x1020 // Date last referenced
	XtuINMLSIZE XmitTextUnitId = 0x8018 // Data set size in megabytes
	XtuINMMEMBR XmitTextUnitId = 0x0003 // Member name list
	XtuINMNUMF  XmitTextUnitId = 0x102F // Number of files transmitted
	XtuINMRECCT XmitTextUnitId = 0x102A // Transmitted record count
	XtuINMRECFM XmitTextUnitId = 0x0049 // Record format
	XtuINMSECND XmitTextUnitId = 0x000B // Secondary space quantity
	XtuINMSIZE  XmitTextUnitId = 0x102C // File size in bytes
	XtuINMTERM  XmitTextUnitId = 0x0028 // Data transmitted as a message
	XtuINMTNODE XmitTextUnitId = 0x1001 // Target node name or node number
	XtuINMTTIME XmitTextUnitId = 0x1025 // Destination timestamp
	XtuINMTUID  XmitTextUnitId = 0x1002 // Target user ID
	XtuINMTYPE  XmitTextUnitId = 0x8012 // Data set type
	XtuINMUSERP XmitTextUnitId = 0x1029 // User parameter string
	XtuINMUTILN XmitTextUnitId = 0x1028 // Name of utility program
)

type XmitTextUnitData struct {
	Len  uint16
	Data []byte
}

type XmitTextUnit interface {
	Id() XmitTextUnitId
	Count() uint16
	Data() []XmitTextUnitData
}

type XmitTextUnitImpl struct {
	IdValue    XmitTextUnitId
	CountValue uint16
	DataValue  []XmitTextUnitData
}

func newXmitTextUnit(raw []byte) (XmitTextUnit, int) {
	numbytes := 0
	if len(raw) < 4 {
		return nil, 0 // Not enough data to form a valid XmitTextUnit
	}
	// Build a byte buffer from the raw data
	buf := bytes.NewBuffer(raw)
	// Extract the first 2 bytes as the ID. They are in big-endian format.
	id := XmitTextUnitId(binary.BigEndian.Uint16(buf.Next(2)))
	// Do the same for the count, which is the next 2 bytes.
	count := binary.BigEndian.Uint16(buf.Next(2))
	// Allocate a slice of XmitTextUnitData with the size of the count
	data := make([]XmitTextUnitData, count)
	numbytes += 4 // 2 bytes for ID and 2 bytes for count
	// Read the data for each XmitTextUnitData
	for i := uint16(0); i < count; i++ {
		if len(buf.Bytes()) < 2 {
			return nil, 0 // Not enough data for the next XmitTextUnitData
		}
		// Read the length of the data
		dataLen := binary.BigEndian.Uint16(buf.Next(2))
		if len(buf.Bytes()) < int(dataLen) {
			return nil, 0 // Not enough data for the full XmitTextUnitData
		}
		data[i] = XmitTextUnitData{
			Len:  dataLen,
			Data: buf.Next(int(dataLen)),
		}
		numbytes += 2 + int(dataLen) // 4 bytes for the length of the data and the data itself
	}
	return &XmitTextUnitImpl{
		IdValue:    id,
		CountValue: count,
		DataValue:  data,
	}, numbytes
}

// Implement XmitTextUnit interface for XmitTextUnitImpl

func (x *XmitTextUnitImpl) Id() XmitTextUnitId {
	return x.IdValue
}

func (x *XmitTextUnitImpl) Count() uint16 {
	return x.CountValue
}

func (x *XmitTextUnitImpl) Data() []XmitTextUnitData {
	return x.DataValue
}
