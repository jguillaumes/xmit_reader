package xmitfile

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/jguillaumes/go-ebcdic"
	xu "github.com/jguillaumes/xmit_reader/internal/xmitutils"
)

type XmitFileParams struct {
	sourceDDName   string
	sourceDSName   string
	sourceDsorg    string
	sourceDstype   string
	sourceCreation time.Time
	sourceRecfm    string
	sourceLrecl    int16
	sourceBlksize  int16
	aproxSize      int64
	utilPgmName    string
}

type XmitParams struct {
	sourceNodeName string
	sourceUserId   string
	sourceTstamp   time.Time
	numFiles       int
	XmitFiles      []XmitFileParams
}

func NewXmitParams() *XmitParams {
	return &XmitParams{
		sourceNodeName: "",
		sourceUserId:   "",
		numFiles:       0,
		XmitFiles:      make([]XmitFileParams, 0),
	}
}

func ProcessXMITFile(inFile io.Reader, targetDir string) (int, error) {

	count := 0
	data, err := readXMITRecord(inFile)
	xmitParms := *NewXmitParams()
	var endOfXmit bool = false

	for data != nil && err == nil && endOfXmit == false {
		fmt.Printf("Record Length: %3d, flags: %08b, id: %s\n", data.recordLen(), data.recordFlags(), data.recordId())
		switch data.recordId() {
		case "INMR01":
			tus := data.textUnits(0)
			for t := range tus {
				tu := tus[t]
				switch tu.Id() {
				case XtuINMNUMF:
					xtud := tu.Data()[0]
					nb := xtud.Len
					xmitParms.numFiles = xu.GetVariableLengthInt(int(nb), xtud.Data)
				case XtuINMFUID:
					userId, _ := ebcdic.Decode(tu.Data()[0].Data, ebcdic.EBCDIC037)
					xmitParms.sourceUserId = userId
				case XtuINMFNODE:
					nodeName, _ := ebcdic.Decode(tu.Data()[0].Data, ebcdic.EBCDIC037)
					xmitParms.sourceNodeName = nodeName
				case XtuINMFTIME:
					tuDv := tu.Data()[0]
					// The timestamp is in the format YYYYMMDDHHMSS in EBCDIC
					tstamp, _ := ebcdic.Decode(tuDv.Data, ebcdic.EBCDIC037)
					// Parse the timestamp
					timestamp, err := time.Parse("20060102150405", tstamp)
					if err != nil {
						fmt.Printf("Error parsing timestamp: %v\n", err)
					} else {
						xmitParms.sourceTstamp = timestamp
					}
				}
			}
		case "INMR02":
			var fileParams XmitFileParams
			filenumber := binary.BigEndian.Uint32(data.recordData()[6:10])
			fmt.Printf("File Number: %d\n", filenumber)
			tus := data.textUnits(4)
			for t := range tus {
				tu := tus[t]
				switch tu.Id() {
				case XtuINMUTILN:
					utilPgmName, _ := ebcdic.Decode(tu.Data()[0].Data, ebcdic.EBCDIC037)
					fileParams.utilPgmName = utilPgmName
				case XtuINMDSORG:
					dsorgBytes := xu.GetVariableLengthInt(2, tu.Data()[0].Data)
					switch dsorgBytes {
					case 0x0008:
						fileParams.sourceDsorg = "VSAM"
					case 0x0200:
						fileParams.sourceDsorg = "PO"
					case 0x4000:
						fileParams.sourceDsorg = "PS"
					default:
						fileParams.sourceDsorg = "UNKNOWN"
					}
				case XtuINMTYPE:
					tuDv := tu.Data()[0]
					dstyteByte := tuDv.Data[0]
					switch dstyteByte {
					case 0x80:
						fileParams.sourceDstype = "LIBRARY"
					case 0x40:
						fileParams.sourceDstype = "PGMLIB"
					case 0x04:
						fileParams.sourceDstype = "EXTENDED"
					case 0x01:
						fileParams.sourceDstype = "LARGE"
					}
				case XtuINMRECFM:
					tuDv := tu.Data()[0]
					recfmBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					var fixed = ""
					var variable = ""
					var ctlasa = ""
					var blocked = ""
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
					fileParams.sourceRecfm = fmt.Sprintf("%s%s%s%s", fixed, variable, blocked, ctlasa)
				case XtuINMLRECL:
					tuDv := tu.Data()[0]
					lreclBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					fileParams.sourceLrecl = int16(lreclBytes)
				case XtuINMBLKSZ:
					tuDv := tu.Data()[0]
					blksizeBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					fileParams.sourceBlksize = int16(blksizeBytes)
				case XtuINMSIZE:
					tuDv := tu.Data()[0]
					aproxSizeBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					fileParams.aproxSize = int64(aproxSizeBytes)
				case XtuINMDDNAM:
					ddname, _ := ebcdic.Decode(tu.Data()[0].Data, ebcdic.EBCDIC037)
					fileParams.sourceDDName = ddname
				case XtuINMDSNAM:
					parts := tus[t].Count()
					var dsname string
					for i := uint16(0); i < parts; i++ {
						partData := tus[t].Data()[i]
						partName, _ := ebcdic.Decode(partData.Data, ebcdic.EBCDIC037)
						dsname += partName
						if i < parts-1 {
							dsname += "."
						}
					}
					fileParams.sourceDSName = dsname
				}
			}
			xmitParms.XmitFiles = append(xmitParms.XmitFiles, fileParams)
		case "INMR06": // Last record in the XMIT file, end processing here
			fmt.Println("End of XMIT file processing.")
			endOfXmit = true
		}
		count++
		data, err = readXMITRecord(inFile)
	}
	fmt.Printf("XMIT parameters: %+v\n", xmitParms)

	// Discard err if it is EOF
	if err != nil && err == io.EOF {
		err = nil
	}

	return count, err
}
