// Copyright 2012 Lawrence Kesteloot

package main

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"fmt"
	"headcode.com/webutil"
	"log"
	"net/http"
	"time"
)

// Simple interface that has a Timeout() method, since so many net errors have
// the method.

type Timeouter interface {
	Timeout() bool
}

func generateIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	bw := bufio.NewWriter(w)
	fmt.Fprint(bw, `<!DOCTYPE html>
<html>
    <head>
        <title>TRS-80 Model III Emulator</title>
        <script src="static/jquery-1.8.2.min.js"></script>
        <script src="static/home.js"></script>
        <link rel="stylesheet" href="static/home.css"/>
        <link rel="stylesheet" href="font.css"/>
	</head>
	<body>
	</body>
</html>`)
	bw.Flush()
}

func generateFontCss(w http.ResponseWriter, r *http.Request) {
	// Image is 512x480
	// 10 rows of glyphs, but last two are different page.
	// Use first 8 rows.
	// 32 chars across (32*8 = 256)
	// For thin font:
	//     256px wide.
	//     Chars are 8px wide (256/32 = 8)
	//     Chars are 24px high (480/2/10 = 24), with doubled rows.
	w.Header().Set("Content-Type", "text/css")
	bw := bufio.NewWriter(w)
	fmt.Fprint(bw, `.char {
		display: inline-block;
		width: 8px;
		height: 24px;
		background-image: url("static/font.png");
		/* background-position: -248px -24px;*/ /* ? = 31*8, 1*24 */
		background-position: 0 0; /* Blank */
		background-repeat: no-repeat;
}
`)
	for ch := 0; ch < 256; ch++ {
		fmt.Fprintf(bw, ".char-%d { background-position: %dpx %dpx; }\n",
			ch, -(ch%32)*8, -(ch/32)*24)
	}
	bw.Flush()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		generateIndex(w, r)
	} else if r.URL.Path == "/font.css" {
		generateFontCss(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// Goroutine to read from the ws and send us the commands.
func readWs(ws *websocket.Conn, vmCommandCh chan<- vmCommand) {
	for {
		var message vmCommand

		err := websocket.JSON.Receive(ws, &message)
		if err != nil {
			timeoutErr, ok := err.(Timeouter)
			if ok && timeoutErr.Timeout() {
				// Timeout okay, just retry.
				continue
			}
			log.Printf("websocket.JSON.Receive: %s", err)
			vmCommandCh <- vmCommand{Cmd: "shutdown"}
			return
		}
		log.Printf("Got command %s", message)
		vmCommandCh <- message
	}
}

func wsHandler(ws *websocket.Conn) {
	vmCommandCh := make(chan vmCommand)
	vmUpdateCh := make(chan vmUpdate)
	go readWs(ws, vmCommandCh)
	vm := createVm(vmUpdateCh)
	go vm.run(vmCommandCh)

	// Batch updates.
	var vmUpdates []vmUpdate
	flushUpdates := func() bool {
		if len(vmUpdates) > 0 {
			/// log.Printf("Flushing %d updates", len(vmUpdates))
			err := websocket.JSON.Send(ws, vmUpdates)
			if err != nil {
				log.Printf("websocket.JSON.Send: %s", err)
				return false
			}
			// Clear queue.
			vmUpdates = vmUpdates[:0]
		}

		return true
	}
	tickerCh := time.Tick(10 * time.Millisecond)

	receiving := true
	for receiving {
		select {
		case update := <-vmUpdateCh:
			vmUpdates = append(vmUpdates, update)
			if update.Cmd == "shutdown" {
				flushUpdates()
				receiving = false
			}
		case <-tickerCh:
			receiving = flushUpdates()
		}
	}
}

func serveWebsite() {
	port := 8080

	// Create handlers.
	handlers := http.NewServeMux()
	handlers.Handle("/", webutil.GetHandler(http.HandlerFunc(homeHandler)))
	handlers.Handle("/ws", websocket.Handler(wsHandler))
	handlers.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// Create server.
	address := fmt.Sprintf(":%d", port)
	server := http.Server{
		Addr:           address,
		Handler:        webutil.LoggingHandler(handlers),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: http.DefaultMaxHeaderBytes,
	}

	// Start serving.
	log.Printf("Serving website on %s", address)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
