package main

// http://www.trs-80.com/trs80-zaps-internals.htm#portsm3
// http://www.trs-80.com/trs80-zaps-internals.htm#ports
var ports map[byte]string = map[byte]string{
	0xFF: "cassette port",
}
