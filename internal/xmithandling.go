package internal

import (
	"bufio"
	"encoding/binary"
	"fmt"

	"github.com/jguillaumes/go-ebcdic"
)

func ProcessXMITFile(inFile *bufio.Reader, targetDir string) (int, error) {

	count := 0
	data, err := readXMITRecord(inFile)

	for data != nil && err == nil {
		fmt.Printf("Record Length: %3d, flags: %08b, id: %s\n", data.recordLen(), data.recordFlags(), data.recordId())
		switch data.recordId() {
		case "INMR01":
			tus := data.textUnits(0)
			for t := range tus {
				tu := tus[t]
				switch tu.Id() {
				case XtuINMLRECL:
					lrecl_raw := tu.Data()[0].Data
					var lrecl int
					if len(lrecl_raw) == 1 {
						lrecl = int(lrecl_raw[0])
					} else if len(lrecl_raw) == 2 {
						lrecl = int(binary.BigEndian.Uint16(lrecl_raw))
					} else {
						lrecl = -1 // Invalid length
					}
					fmt.Printf("Logical Record Length: %d\n", lrecl)
				case XtuINMFNODE:
					nodeName, _ := ebcdic.Decode(tu.Data()[0].Data, ebcdic.EBCDIC037)
					fmt.Printf("Node Name: %s\n", nodeName)
				default:
					fmt.Printf("id: %04x\n", tus[t].Id())
				}
			}
		case "INMR02":
			filenumber := binary.BigEndian.Uint32(data.recordData()[6:10])
			fmt.Printf("File Number: %d\n", filenumber)
			tus := data.textUnits(4)
			for t := range tus {
				switch tus[t].Id() {
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
					fmt.Printf("Data Set Name: %s\n", dsname)
				default:
					// For any other text unit, just print the ID
					fmt.Printf("id: %04x\n", tus[t].Id())
				}
			}
		}
		count++
		data, err = readXMITRecord(inFile)
	}
	// Discard err if it is EOF
	if err != nil && err.Error() == "EOF" {
		err = nil
	}

	return count, err
}
