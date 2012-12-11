package main

import (
	"bufio"
	"net/http"
	"time"
	"log"
	"fmt"
	"headcode.com/webutil"
	"code.google.com/p/go.net/websocket"
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
	</head>
	<body>
	</body>
</html>`)
	bw.Flush()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		generateIndex(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func wsHandler(ws *websocket.Conn, ch <-chan cpuUpdate) {
	log.Printf("wsHandler")

	for update := range ch {
		websocket.JSON.Send(ws, update)
	}
}

func serveWebsite(ch <-chan cpuUpdate) {
	port := 8080

	// Create handlers.
	handlers := http.NewServeMux()
	handlers.Handle("/", webutil.GetHandler(http.HandlerFunc(homeHandler)))
	handlers.Handle("/ws", websocket.Handler(func (ws *websocket.Conn) {
		wsHandler(ws, ch)
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
