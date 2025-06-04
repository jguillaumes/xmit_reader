package main

import (
	"flag"
	"log"
	"os"

	"github.com/jguillaumes/xmit_reader/internal/unloadfile"
	"github.com/jguillaumes/xmit_reader/internal/xmitfile"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
		log.Println("Target directory does not exist:", *targetDir)
		os.Exit(1)
	}

	// Check if an unload file is specified. Id so, open it for write
	// otherwise, create a temporary file
	if *unloadFile == "" {
		tempFile, err := os.CreateTemp("", "xmit_unload_*.unload")
		if err != nil {
			log.Println("Error creating temporary unload file:", err.Error())
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
		log.Println("Error opening unload file:", err.Error())
		os.Exit(1)
	}

	// Open the input file
	inFile, err := os.Open(*inputFile)
	if err != nil {
		log.Println("Error opening input file:", err.Error())
		os.Exit(1)
	}
	defer inFile.Close()

	// Process the input file and generate output files
	var count int
	count, err = xmitfile.ProcessXMITFile(inFile, *targetDir, unloadFileHandle)
	if err != nil {
		log.Println("Error processing input file:", err.Error())
		os.Exit(1)
	}

	if count == 0 {
		log.Println("No members extracted from the input file.")
	} else {
		log.Println("Successfully processed", count, "records from the XMIT input file.")
	}

	// Close the unload file handle
	if err := unloadFileHandle.Close(); err != nil {
		log.Println("Error closing unload file:", err.Error())
		os.Exit(1)
	}
	// Reopen the unload file to read its contents
	unloadFileHandle, err = os.Open(*unloadFile)
	if err != nil {
		log.Println("Error reopening unload file for reading:", err.Error())
		os.Exit(1)
	}
	defer unloadFileHandle.Close()

	_, err = unloadfile.ProcessUnloadFile(*unloadFileHandle, *targetDir)

	if deleteUnloadFile {
		// Delete the unload file if it was created as a temporary file
		if err := os.Remove(*unloadFile); err != nil {
			log.Println("Error deleting unload file:", err.Error())
		} else {
			log.Println("Temporary unload file deleted:", *unloadFile)
		}
	}

	os.Exit(0)
}
