// Copyright 2012 Lawrence Kesteloot

package main

// Expose a web interface for the UI of the machine.

import (
	"bufio"
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"github.com/lkesteloot/goutil/webutil"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Simple interface that has a Timeout() method, since so many net errors have
// the method.
type Timeouter interface {
	Timeout() bool
}

// Generate the top-level index page.
func generateIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, "static/index.html")
}

// Generate the font CSS that has offsets for each character.
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

// Generate a JSON document of files in a directory.
func generateFileList(w http.ResponseWriter, r *http.Request, dir string) {
	// Get list of files.
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Convert to list of strings.
	var filenames []string
	for _, fileInfo := range fileInfos {
		filenames = append(filenames, fileInfo.Name())
	}

	// JSON-encoded.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filenames)
}

// Top-level handler.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		generateIndex(w, r)
	case "/font.css":
		generateFontCss(w, r)
	case "/disks.json":
		generateFileList(w, r, "disks")
	case "/cassettes.json":
		generateFileList(w, r, "cassettes")
	default:
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
		/// log.Printf("Got command %s", message)
		vmCommandCh <- message
	}
}

// Handle the web sockets request.
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
			// Combine consecutive pokes.
			last := len(vmUpdates) - 1
			if update.Cmd == "poke" && last >= 0 && vmUpdates[last].Cmd == "poke" &&
				vmUpdates[last].Addr+len(vmUpdates[last].Msg) == update.Addr {

				// Just tack it on to the existing poke.
				vmUpdates[last].Msg += update.Msg
			} else {
				// Add update to queue.
				vmUpdates = append(vmUpdates, update)
			}
			if update.Cmd == "shutdown" {
				flushUpdates()
				receiving = false
			}
		case <-tickerCh:
			receiving = flushUpdates()
		}
	}
}

// Serve the website. This function blocks.
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
