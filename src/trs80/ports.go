package main

// http://www.trs-80.com/trs80-zaps-internals.htm#portsm3
// http://www.trs-80.com/trs80-zaps-internals.htm#ports
var ports map[byte]string = map[byte]string{
	0xE0: "maskable interrupt",
	0xE4: "NMI options/status",
	0xEC: "various controls",
	0xF0: "FDC command/status",
	0xF4: "select drive and options",
	0xFF: "cassette port",
}
