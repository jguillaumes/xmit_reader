package xmitfile

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	e "github.com/jguillaumes/go-encoding/encodings"
	xu "github.com/jguillaumes/xmit_reader/internal/xmitutils"
)

var enc = e.NewEncoding()

type XmitFileParams struct {
	SourceDDName   string    `json:"ddame"`
	SourceDSName   string    `json:"dsname"`
	SourceDsorg    string    `json:"dssorg"`
	SourceDstype   string    `json:"dsstype"`
	SourceCreation time.Time `json:"creation"`
	SourceRecfm    string    `json:"recfm"`
	SourceLrecl    int16     `json:"lrecl"`
	SourceBlksize  int16     `json:"blksize"`
	AproxSize      int64     `json:"aprox_size"`
	UtilPgmName    string    `json:"util_pgm_name"`
}

type XmitParams struct {
	SourceNodeName string           `json:"source_node_name"`
	SourceUserId   string           `json:"source_user_id"`
	SourceTstamp   time.Time        `json:"source_timestamp"`
	NumFiles       int              `json:"num_files"`
	XmitFiles      []XmitFileParams `json:"xmit_files"`
}

func NewXmitParams() *XmitParams {
	return &XmitParams{
		SourceNodeName: "",
		SourceUserId:   "",
		NumFiles:       0,
		XmitFiles:      make([]XmitFileParams, 0),
	}
}

func ProcessXMITFile(inFile io.Reader, targetDir string, unloadFile io.Writer, encoding string) (*XmitParams, error) {

	count := 0
	xmitParms := *NewXmitParams()
	var endOfXmit bool = false
	var currentBlock *bytes.Buffer

	for !endOfXmit {
		data, err := readXMITRecord(inFile)
		if err != nil {
			return nil, err
		}
		log.Debugf("Record Length: %3d, flags: %08b, id: %s\n", data.recordLen(), data.recordFlags(), data.recordId())
		switch data.recordId() {
		case "INMR01":
			tus := data.textUnits(0)
			for t := range tus {
				tu := tus[t]
				switch tu.Id() {
				case XtuINMNUMF:
					xtud := tu.Data()[0]
					nb := xtud.Len
					xmitParms.NumFiles = xu.GetVariableLengthInt(int(nb), xtud.Data)
				case XtuINMFUID:
					userId, _ := enc.DecodeBytes(tu.Data()[0].Data, encoding)
					xmitParms.SourceUserId = userId
				case XtuINMFNODE:
					nodeName, _ := enc.DecodeBytes(tu.Data()[0].Data, encoding)
					xmitParms.SourceNodeName = nodeName
				case XtuINMFTIME:
					tuDv := tu.Data()[0]
					// The timestamp is in the format YYYYMMDDHHMSS in EBCDIC
					tstamp, _ := enc.DecodeBytes(tuDv.Data, encoding)
					// Parse the timestamp
					timestamp, err := time.Parse("20060102150405", tstamp)
					if err != nil {
						log.Warnf("Error parsing timestamp: %v\n", err)
					} else {
						xmitParms.SourceTstamp = timestamp
					}
				}
			}
		case "INMR02":
			var fileParams XmitFileParams
			_ = binary.BigEndian.Uint32(data.recordData()[6:10])
			tus := data.textUnits(4)
			for t := range tus {
				tu := tus[t]
				switch tu.Id() {
				case XtuINMUTILN:
					utilPgmName, _ := enc.DecodeBytes(tu.Data()[0].Data, "IBM-1047")
					fileParams.UtilPgmName = utilPgmName
				case XtuINMDSORG:
					dsorgBytes := xu.GetVariableLengthInt(2, tu.Data()[0].Data)
					switch dsorgBytes {
					case 0x0008:
						fileParams.SourceDsorg = "VSAM"
					case 0x0200:
						fileParams.SourceDsorg = "PO"
						fileParams.SourceDstype = "PDS"
					case 0x4000:
						fileParams.SourceDsorg = "PS"
					default:
						fileParams.SourceDsorg = "UNKNOWN"
					}
				case XtuINMTYPE:
					tuDv := tu.Data()[0]
					dstyteByte := tuDv.Data[0]
					switch dstyteByte {
					case 0x80:
						fileParams.SourceDstype = "LIBRARY"
					case 0x40:
						fileParams.SourceDstype = "PGMLIB"
					case 0x04:
						fileParams.SourceDstype = "EXTENDED"
					case 0x01:
						fileParams.SourceDstype = "LARGE"
					}
				case XtuINMRECFM:
					tuDv := tu.Data()[0]
					recfmBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					fileParams.SourceRecfm = xu.RecfmHwToString(uint16(recfmBytes))
				case XtuINMCREAT:
					tuDv := tu.Data()[0]
					// The creation date is in the format YYYYMMDD in EBCDIC
					creationDate, _ := enc.DecodeBytes(tuDv.Data, "IBM-1047")
					// Parse the creation date
					creation, _ := time.Parse("20060102", creationDate)
					fileParams.SourceCreation = creation
				case XtuINMLRECL:
					tuDv := tu.Data()[0]
					lreclBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					fileParams.SourceLrecl = int16(lreclBytes)
				case XtuINMBLKSZ:
					tuDv := tu.Data()[0]
					blksizeBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					fileParams.SourceBlksize = int16(blksizeBytes)
				case XtuINMSIZE:
					tuDv := tu.Data()[0]
					aproxSizeBytes := xu.GetVariableLengthInt(int(tuDv.Len), tuDv.Data)
					fileParams.AproxSize = int64(aproxSizeBytes)
				case XtuINMDDNAM:
					ddname, _ := enc.DecodeBytes(tu.Data()[0].Data, encoding)
					fileParams.SourceDDName = ddname
				case XtuINMDSNAM:
					parts := tus[t].Count()
					var dsname string
					for i := uint16(0); i < parts; i++ {
						partData := tus[t].Data()[i]
						partName, _ := enc.DecodeBytes(partData.Data, encoding)
						dsname += partName
						if i < parts-1 {
							dsname += "."
						}
					}
					fileParams.SourceDSName = dsname
				default:
					log.Tracef("Unknown text unit ID: %04x\n", tu.Id())
				}
			}
			xmitParms.XmitFiles = append(xmitParms.XmitFiles, fileParams)
		case "INMR03":
			// File header record, ignore it
		case "INMR04":
			// User control record, ignore it
		case "INMR06": // Last record in the XMIT file, end processing here
			log.Debugln("End of XMIT file processing.")
			endOfXmit = true
		case "INMR07":
			// Notification record, ignore it
		default:
			// Data reecord
			if data.recordFlags()&FirstSegment != 0 {
				currentBlock = bytes.NewBuffer(make([]byte, 0, 32767))
			}
			currentBlock.Write(data.recordData())
			if data.recordFlags()&LastSegment != 0 {
				blockLen := int16(currentBlock.Len()) + 8
				lenBytes := make([]byte, 8)
				binary.BigEndian.PutUint16(lenBytes, uint16(blockLen))
				binary.BigEndian.PutUint16(lenBytes[2:], uint16(0))
				binary.BigEndian.PutUint32(lenBytes[4:], uint32(0))
				unloadFile.Write(lenBytes)
				unloadFile.Write(currentBlock.Bytes())
			}
		}
		count++
	}
	if log.GetLevel() >= log.DebugLevel {
		marshalled, err := json.MarshalIndent(xmitParms, "", "  ")
		if err != nil {
			log.Warnf("Error marshalling XMIT parameters: %v\n", err)
		} else {
			log.Debugf("XMIT parameters: %s\n", marshalled)
		}
	}

	return &xmitParms, nil
}
