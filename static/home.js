
(function () {

    var msgCount = 0;

    var createScreen = function () {
        var $screen = $("<div>").addClass("screen").appendTo($("body"));

        var addr = 15360;
        for (var y = 0; y < 16; y++) {
            for (var x = 0; x < 64; x++) {
                var $ch = $("<span>").attr("id", "s" + addr).addClass("char");
                $screen.append($ch);
                addr++;
            }
            $screen.append($("<br>"));
        }
    };

    var handleMsg = function (msg) {
        var cmd = msg.Cmd;

        if (cmd === "poke") {
            var addr = msg.Addr;
            var data = msg.Data;

            if (addr >= 15360 && addr < 16384) {
                // Screen.
                $("#s" + addr).attr("class", "char char-" + data);
            }
        } else {
            console.log("Unknown command \"" + cmd + "\"");
        }

        msgCount++;
        if (msgCount % 1000 === 0) {
            console.log("Got " + msgCount + " messages")
        }
    };

    var configureConnection = function () {
        console.log("Creating web socket");
        var ws = new WebSocket("ws://" + window.location.host + "/ws");
        ws.onopen = function (event) {
            console.log("On open");
            ws.send("Hello!");
        };
        ws.onmessage = function (event) {
            // https://developer.mozilla.org/en-US/docs/WebSockets/WebSockets_reference/MessageEvent
            var msg = JSON.parse(event.data);
            handleMsg(msg);
        };
        ws.onclose = function (event) {
            // https://developer.mozilla.org/en-US/docs/WebSockets/WebSockets_reference/CloseEvent
            console.log("On close (" + event.code + ")");
        };
    };

    $(function () {
        createScreen();
        configureConnection();
    });
})();
