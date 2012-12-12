
(function () {

    var msgCount = 0;
    var commandWs = null;

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

    var createButtons = function () {
        var $buttons = $("<div>").addClass("buttons").appendTo($("body"));

        $("<button>").
            attr("type", "button").
            text("Boot").
            click(function () {
                if (commandWs) {
                    commandWs.send(JSON.stringify({Cmd: "boot"}));
                }
            }).
            appendTo($buttons);
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

    var configureUpdates = function () {
        var ws = new WebSocket("ws://" + window.location.host + "/updates.ws");
        ws.onopen = function (event) {
            // Nothing.
        };
        ws.onmessage = function (event) {
            // https://developer.mozilla.org/en-US/docs/WebSockets/WebSockets_reference/MessageEvent
            var msg = JSON.parse(event.data);
            handleMsg(msg);
        };
        ws.onclose = function (event) {
            // https://developer.mozilla.org/en-US/docs/WebSockets/WebSockets_reference/CloseEvent
            console.log("On update close (" + event.code + ")");
            commandWs = null;
        };
    };

    var configureCommands = function () {
        var ws = new WebSocket("ws://" + window.location.host + "/commands.ws");
        ws.onopen = function (event) {
            // Nothing.
        };
        ws.onmessage = function (event) {
            // Nothing.
        };
        ws.onclose = function (event) {
            // https://developer.mozilla.org/en-US/docs/WebSockets/WebSockets_reference/CloseEvent
            console.log("On command close (" + event.code + ")");
        };
        return ws;
    };

    var configureKeyboard = function () {
        var keyEvent = function (event, isPressed) {
            var ch = event.which;

            console.log(ch);
            if (ch === 13) {
                // Enter.
                ch = 48;
            } else if (ch === 32) {
                // Space.
                ch = 55;
            } else if (ch === 8) {
                // Backspace.
                ch = 53; // Left.

                // Don't go back to previous page.
                event.preventDefault();
            } else if (ch >= 65 && ch <= 90) {
                // Letters, convert to 1-26.
                ch -= 64;
            } else if (ch >= 48 && ch <= 57) {
                // Letters, convert to 32-41.
                ch -= 16;
            } else {
                // Ignore.
                ch = -1;
            }

            if (ch !== -1 && commandWs) {
                commandWs.send(JSON.stringify({
                    Cmd: isPressed ? "press" : "release",
                    Data: ch
                }));
            }
        };

        $("body").keydown(function (event) {
            keyEvent(event, true);
        }).keyup(function (event) {
            keyEvent(event, false);
        });
    };

    $(function () {
        createScreen();
        createButtons();
        configureUpdates();
        commandWs = configureCommands();
        configureKeyboard();
    });
})();
