This is a TRS-80 Model III emulator written in Go. It uses a web page for its
interface. It can read diskettes and cassettes. It uses the
[Z80 emulation library](https://github.com/remogatto/z80) from
[Andrea Fazzi](https://plus.google.com/u/0/100271912081202470197/about).

Installing
----------

Install with:

    go get github.com/lkesteloot/trs80emu

Go to the source directory, which is probably:

    $GOPATH/src/github.com/lkesteloot/trs80emu

(If you have more than one item in your $GOPATH, use only the first one.)

Run with:

    ../../../../bin/trs80emu

or just:

    ./GO

Then go to this address with your web browser:

    http://localhost:8080/

and click the Boot button.

Diskettes
---------

You can change the contents of the disk drives with the selectors
on the right. The red dots represent the drive motors. A few diskettes
are included with the source. Add more into the "disks" directory.
Only reading is implemented. All diskettes pretend to be write-protected.

Cassettes
---------

You can change the contents of the cassette with the selector on the right.
The red dot represents the cassette motor. Put the cassette files into the
"cassettes" directory.  Cassettes must be WAV files (mono, 16-bit). Both 500
and 1500 baud are supported.

Screenshots
-----------

TRS-DOS:

![TRS-DOS](https://raw.github.com/lkesteloot/trs80emu/master/screenshots/02_disk_boot.png)

VisiCalc:

![VisiCalc](https://raw.github.com/lkesteloot/trs80emu/master/screenshots/03_visicalc.png)

Loading from cassette:

![Loading from cassette](https://raw.github.com/lkesteloot/trs80emu/master/screenshots/04_cload.png)

Eliminator game (on aigames1.dsk):

![Eliminator](https://raw.github.com/lkesteloot/trs80emu/master/screenshots/05_eliminator.png)
