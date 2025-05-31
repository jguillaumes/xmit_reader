package main

import (
	"flag"
	"os"

	"github.com/jguillaumes/xmit_reader/internal/xmitfile"
)

func main() {
	// Command line arguments:
	// --input <input_file>: The input XMIT file to process.
	// --target <target_directory>: The directory where the output files will be saved.

	inputFile := flag.String("input", "", "L'arxiu XMIT d'entrada a processar")
	targetDir := flag.String("target", "", "Directori on es guardaran els arxius de sortida")
	unloadFile := flag.String("unload", "", "Arxiu de descarrega (opcional). Si no s'especifica, s'usarà un arxiu temporal")

	flag.Parse()

	var deleteUnloadFile bool = true

	// Check if input file and target directory are provided
	if *inputFile == "" || *targetDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Check if the targert directory exists
	if _, err := os.Stat(*targetDir); os.IsNotExist(err) {
		println("Target directory does not exist:", *targetDir)
		os.Exit(1)
	}

	// Check if an unload file is specified. Id so, open it for write
	// otherwise, create a temporary file
	if *unloadFile == "" {
		tempFile, err := os.CreateTemp("", "xmit_unload_*.unload")
		if err != nil {
			println("Error creating temporary unload file:", err.Error())
			os.Exit(1)
		}
		defer tempFile.Close()
		*unloadFile = tempFile.Name()
		deleteUnloadFile = true
	} else {
		deleteUnloadFile = false
	}

	// Unconditionally open the unload file for writing
	unloadFileHandle, err := os.OpenFile(*unloadFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		println("Error opening unload file:", err.Error())
		os.Exit(1)
	}

	// Open the input file
	inFile, err := os.Open(*inputFile)
	if err != nil {
		println("Error opening input file:", err.Error())
		os.Exit(1)
	}
	defer inFile.Close()

	// Process the input file and generate output files
	var count int
	count, err = xmitfile.ProcessXMITFile(inFile, *targetDir, unloadFileHandle)
	if err != nil {
		println("Error processing input file:", err.Error())
		os.Exit(1)
	}

	if count == 0 {
		println("No members extracted from the input file.")
	} else {
		println("Successfully processed", count, "records from the XMIT input file.")
	}

	if deleteUnloadFile {
		// Delete the unload file if it was created as a temporary file
		if err := os.Remove(*unloadFile); err != nil {
			println("Error deleting unload file:", err.Error())
		} else {
			println("Temporary unload file deleted:", *unloadFile)
		}
	}

	os.Exit(0)
}
