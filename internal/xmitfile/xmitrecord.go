package xmitfile

import (
	"io"

	log "github.com/sirupsen/logrus"
)

type XMITRecordFlags byte

const (
	FirstSegment    XMITRecordFlags = 0x80
	LastSegment     XMITRecordFlags = 0x40
	IsControlRecord XMITRecordFlags = 0x20
	IsRecordNumber  XMITRecordFlags = 0x10
)

type XMITRecord interface {
	recordLen() byte
	recordFlags() XMITRecordFlags
	recordData() []byte
	recordId() string
	textUnits(uint8) []XmitTextUnit
}

type XMITRecordImpl struct {
	recordLenValue   byte
	recordFlagsValue XMITRecordFlags
	recordDataValue  []byte
}

func (x *XMITRecordImpl) recordLen() byte              { return x.recordLenValue }
func (x *XMITRecordImpl) recordFlags() XMITRecordFlags { return x.recordFlagsValue }
func (x *XMITRecordImpl) recordData() []byte           { return x.recordDataValue }

func (x *XMITRecordImpl) recordId() string {

	if x.recordFlagsValue&IsControlRecord == 0 {
		// If the record is not a control record, return an empty string
		return ""
	}

	// Take the firtst 6 bytes of the data as a binary slice
	if len(x.recordDataValue) < 6 {
		return ""
	}
	idBytes := x.recordDataValue[:6]

	// Translate from EBCDIC (1047) to ASCII (UTF-8)
	// enc is a module level variable declared in xmithandling.go
	id, err := enc.DecodeBytes(idBytes, "IBM-1047")
	if err != nil {
		return ""
	}
	// Convert the byte slice to a string and return it
	return id
}

func (x *XMITRecordImpl) textUnits(offset uint8) []XmitTextUnit {
	// If the record is not a control record, return an empty slice
	if x.recordFlagsValue&IsControlRecord == 0 {
		return nil
	}
	// Create a vector of XmitTextUnit to hold the text units
	var textUnits []XmitTextUnit

	// Make a slice past the first 6 bytes of the data
	data := x.recordDataValue[6+offset:]
	len := len(data)
	for len > 0 {
		tu, numbytes := newXmitTextUnit(data)
		len -= numbytes
		textUnits = append(textUnits, tu)
		data = data[numbytes:]
	}
	return textUnits
}

// readXMITRecord reads a single XMIT record from the provided bufio.Reader.
func readXMITRecord(f io.Reader) (XMITRecord, error) {
	oneByte := make([]byte, 1)

	_, err := f.Read(oneByte)
	if err != nil {
		return nil, err
	}
	recordLen := oneByte[0]

	_, err = f.Read(oneByte)
	if err != nil {
		return nil, err
	}
	recordFlags := oneByte[0]

	// Read the record data
	data := make([]byte, recordLen-2) // -2 for the length and flags bytes
	if l, err := f.Read(data); err != nil {
		return nil, err
	} else if l < int(recordLen-2) {
		log.Errorf("Expected to read %d bytes, but got %d bytes\n", recordLen-2, l)
		return nil, io.ErrUnexpectedEOF
	}

	return &XMITRecordImpl{
		recordLenValue:   recordLen,
		recordFlagsValue: XMITRecordFlags(recordFlags),
		recordDataValue:  data,
	}, nil
}
