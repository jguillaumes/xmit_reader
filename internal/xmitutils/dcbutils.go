package xmitutils

import "fmt"

func RecfmHwToString(recfmBytes uint16) string {
	var fixed = ""
	var variable = ""
	var ctlasa = ""
	var blocked = ""
	var spanned = ""
	if (recfmBytes & 0x0801) != 0 {
		spanned = "S"
	}
	if recfmBytes&0x1000 != 0 {
		blocked = "B"
	}
	if recfmBytes&0x4000 != 0 {
		variable = "V"
	}
	if recfmBytes&0x8000 != 0 {
		fixed = "F"
	}
	if recfmBytes&0x0400 != 0 {
		ctlasa = "A"
	}
	return fmt.Sprintf("%s%s%s%s%s", fixed, variable, blocked, ctlasa, spanned)
}

func RecfmByteToString(recfmByte byte) string {
	var dcbfmt string
	fmtbits := recfmByte & 0xC0 >> 6 // Mask for the first two bits
	switch fmtbits {
	case 0b11:
		dcbfmt = "U"
	case 0b01:
		dcbfmt = "V"
	case 0b10:
		dcbfmt = "F"
	default:
		dcbfmt = "?"
	}
	if recfmByte&0x10 != 0 {
		dcbfmt += "B" // Indicates that the record is blocked
	}
	if recfmByte&0x08 != 0 {
		dcbfmt += "S" // Indicates that the record is spanned
	}
	asabits := recfmByte & 0x06 >> 1 // Mask for the 5,6 bits
	switch asabits {
	case 0b00:
		// Nothing to do, first byte is part of the data
	case 0b01:
		dcbfmt += "C" // Indicates that the record is IBM carriage control
	case 0b10:
		dcbfmt += "A" // Indicates that the record is ASA control
	}
	return dcbfmt
}
