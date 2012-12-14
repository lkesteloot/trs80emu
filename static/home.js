
(function () {

    var msgCount = 0;
    var g_ws = null;

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
                if (g_ws) {
                    g_ws.send(JSON.stringify({Cmd: "boot"}));
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

    var configureWs = function () {
        var ws = new WebSocket("ws://" + window.location.host + "/ws");
        ws.onmessage = function (event) {
            var msg = JSON.parse(event.data);
            handleMsg(msg);
        };
        ws.onclose = function (event) {
            g_ws = null;
        };
        return ws;
    };

    var configureKeyboard = function () {
        var keyEvent = function (event, isPressed) {
            var ch = event.which;

            console.log(ch);
            // http://www.trs-80.com/trs80-zaps-internals.htm#keyboard13
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
            } else if (ch === 188) {
                // Comma.
                ch = 44;
            } else if (ch === 190) {
                // Period.
                ch = 46;
            } else if (ch >= 65 && ch <= 90) {
                // Letters, convert to 1-26.
                ch -= 64;
            } else if (ch >= 48 && ch <= 57) {
                // Letters, convert to 32-41.
                ch -= 16;
            } else if (ch == 16) {
                // Shift.
                ch = 56;
            } else if (ch == 192) {
                // This is ` on the keyboard, but we translate to @.
                ch = 0;
            } else if (ch == 186) {
                // Semicolon.
                ch = 43;
            } else if (ch == 189) {
                // Hyphen.
                ch = 45;
            } else if (ch == 191) {
                // Slash.
                ch = 47;
            } else if (ch == 37) {
                // Left arrow.
                ch = 53;
            } else if (ch == 39) {
                // Right arrow.
                ch = 54;
            } else if (ch == 40) {
                // Down arrow.
                ch = 52;
            } else if (ch == 38) {
                // Up arrow.
                ch = 50;
            } else {
                // Ignore.
                ch = -1;
            }

            if (ch !== -1 && g_ws) {
                g_ws.send(JSON.stringify({
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
        g_ws = configureWs();
        configureKeyboard();
    });
})();
