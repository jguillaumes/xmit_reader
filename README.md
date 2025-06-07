# MVS TRANSMIT (XMIT) file extractor

This utility can extract the contents (members) of a mainframe PDS or PDSE file packed in XMIT format. The members are written into separate files in the directory specified by the user, who must also choose the extension to be added to the file name.

## Usage

To use this utility you need to have a XMIT file in your workstation.

### Generting a XMIT file from a PDS or PDSE

From a TSO READY prompt of from option 6 in the ISPF menu, issue the following command:

```
[TSO] TRANSMIT NNNN.UUUUUUUU DATASET(<source_pds>) OUTDATASET(<xmit_dataset>)
```

- If you are running this command from ISPF you must write the `TSO` word at the beginning. It is not necessary if you are in the TSO READY prompt.
- `NNNN` and `UUUUUUUU` are, respectively, the destination node and userid for the transmit command. We are not transmitting anything here, so you can put here what you want. The paramerters are mandatory though
- `source_pds` is the name of the source library. The usual ISPF dataset prefix name rule applies here.
- `xmit_dataset` is the name of the packed XMIT file. 

### Downloading the XMIT file

You must download the XMIT dataset generated in the previous step **in binary mode**, either using FTP, sftp, the ZOSMF API, ZOWE CLI or whatever file transfer method you have available. 

### Invoking the utility

The utility is a simple, self contained executable that must be run from your operating system command line.

```bash
 $ ./xmit_reader 
Usage of ./xmit_reader:
  -debug
        Output debug information (maybe quite verbose)
  -input string
        Input XMIT file to be processed
  -target string
        Path to the output directory
  -trace
        Maximum debug output. VERY verbose
  -type string
        File type (to be used as extension)
  -unload string
        Name of the IEBCOPY unload file. If not specified it will be not kept and a temporary file will be used
```

Example:

```
$ ./xmit_reader -input data/jgppds.xmit -target work -type pli  
INFO   [0000] Original dataset: JGUILLA.JGP.PLI            
INFO   [0000] Dataset attributes: DSORG=PO, DSTYPE=PDS, RECFM=FB, LRECL=80, BLKSIZE=23440 
INFO   [0000] Writing file work/JGPP600.pli                
INFO   [0000] Writing file work/JGPP802.pli                
INFO   [0000] Writing file work/JGPS011.pli                
INFO   [0000] Writing file work/JGPF020I.pli               
INFO   [0000] Writing file work/JGPF020O.pli               
INFO   [0000] Writing file work/JGPP001.pli                
INFO   [0000] Writing file work/JGPP801.pli                
INFO   [0000] Writing file work/JGPP999.pli                
INFO   [0000] Writing file work/JGPS010.pli           
```

## Building the utility

The utility is written in golang, and can be built using the standard golang toolset. Just clone the github repository  https://gitlab.jguillaumes.dyndns.org/mftools/xmitreader.git to whatever directory you want,  `cd` into that directory and run `go build`. The executable `xmit_reader`should be built at that same directory.

## Known limitations and bugs

- At this moment this is a very preliminary version, and only RECFM=FB files are supported. Eventually, VB files will be supported. There is no plan to support U (LOAD MODULE) files.
- Aliases are not correctly handled. Depending on the order in the PDS directory, they can be ignored or created as if they were the original member (and in this case, the original member will be missing)

## License

This software is licensed using the two clause BSD license. Look at the `LICENSE.txt` file in the project root directory.

