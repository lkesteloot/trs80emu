
(function () {
    var configureConnection = function () {
        console.log("Creating web socket");
        var ws = new WebSocket("ws://" + window.location.host + "/ws");
        ws.onopen = function (event) {
            console.log("On open");
            ws.send("Hello!");
        };
        ws.onmessage = function (event) {
            // https://developer.mozilla.org/en-US/docs/WebSockets/WebSockets_reference/MessageEvent
            console.log("On message");
            console.log(event.data);
        };
        ws.onclose = function (event) {
            // https://developer.mozilla.org/en-US/docs/WebSockets/WebSockets_reference/CloseEvent
            console.log("On close (" + event.code + ")");
        };
    };

    $(function () {
        configureConnection();
    });
})();
