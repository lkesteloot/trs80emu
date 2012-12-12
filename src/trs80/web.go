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
	w.Header().Set("Content-Type", "text/css")
	bw := bufio.NewWriter(w)
	fmt.Fprint(bw, `.char {
		display: inline-block;
		width: 8px;
		height: 24px;
		background-image: url("static/TRS80CharacterGen.png");
		background-position: -8px -24px;
		background-repeat: no-repeat;
}
`)
	for ch := 0; ch < 256; ch++ {
		fmt.Fprintf(bw, ".char-%d { background-position: %dpx %dpx; }\n",
			ch, -(ch % 32)*8, -(ch / 32)*24)
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

func wsHandler(ws *websocket.Conn, cmdCh chan<- interface{}) {
	log.Printf("wsHandler")
	updateCh := make(chan cpuUpdate)
	cmdCh <- startUpdates{updateCh}

	for update := range updateCh {
		websocket.JSON.Send(ws, update)
	}

	cmdCh <- stopUpdates{}
}

func serveWebsite(cmdCh chan<- interface{}) {
	port := 8080

	// Create handlers.
	handlers := http.NewServeMux()
	handlers.Handle("/", webutil.GetHandler(http.HandlerFunc(homeHandler)))
	handlers.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		wsHandler(ws, cmdCh)
	}))
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
