
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
            var msgs = JSON.parse(event.data);
            for (var i = 0; i < msgs.length; i++) {
                handleMsg(msgs[i]);
            }
        };
        ws.onclose = function (event) {
            g_ws = null;
        };
        return ws;
    };

    var configureKeyboard = function () {
        // Converts a key up/down event to an ASCII character or string representing
        // key, like "Left".
        var eventToKey = function (event) {
            var key;
            var which = event.which;
            var shifted = event.shiftKey;

            // http://www.trs-80.com/trs80-zaps-internals.htm#keyboard13
            if (which === 13) {
                // Enter.
                key = "Enter";
            } else if (which === 32) {
                // Space.
                key = " ";
            } else if (which >= 65 && which < 65+26) {
                // Letters.
                if (!shifted) {
                    // Make lower case.
                    which += 32;
                }
                key = String.fromCharCode(which);
            } else if (which === 48) {
                key = shifted ? ")" : "0";
            } else if (which === 49) {
                key = shifted ? "!" : "1";
            } else if (which === 50) {
                key = shifted ? "@" : "2";
            } else if (which === 51) {
                key = shifted ? "#" : "3";
            } else if (which === 52) {
                key = shifted ? "$" : "4";
            } else if (which === 53) {
                key = shifted ? "%" : "5";
            } else if (which === 54) {
                key = shifted ? "^" : "6";
            } else if (which === 55) {
                key = shifted ? "&" : "7";
            } else if (which === 56) {
                key = shifted ? "*" : "8";
            } else if (which === 57) {
                key = shifted ? "(" : "9";
            } else if (which === 8) {
                // Backspace.
                key = "Left"; // Left.

                // Don't go back to previous page.
                event.preventDefault();
            } else if (which === 187) {
                // Equal.
                key = shifted ? "+" : "=";
            } else if (which === 188) {
                // Comma.
                key = shifted ? "<" : ",";
            } else if (which === 190) {
                // Period.
                key = shifted ? ">" : ".";
            } else if (which == 16) {
                // Shift.
                key = "Shift";
            } else if (which == 192) {
                // Backtick.
                key = shifted ? "~" : "`";
            } else if (which == 186) {
                // Semicolon.
                key = shifted ? ":" : ";";
            } else if (which == 222) {
                // Quote..
                key = shifted ? "\"" : "'";
            } else if (which == 189) {
                // Hyphen.
                key = shifted ? "_" : "-";
            } else if (which == 191) {
                // Slash.
                key = shifted ? "?" : "/";
            } else if (which == 37) {
                // Left arrow.
                key = "Left";
            } else if (which == 39) {
                // Right arrow.
                key = "Right";
            } else if (which == 40) {
                // Down arrow.
                key = "Down";
            } else if (which == 38) {
                // Up arrow.
                key = "Up";
            } else if (which == 27) {
                // Escape.
                key = "Break";
            } else {
                // Ignore.
                console.log(which);
                key = "";
            }

            return key
        };

        var keyEvent = function (event, isPressed) {
            var key = eventToKey(event);
            console.log("Key is \"" + key + "\"");

            if (key !== "" && g_ws) {
                g_ws.send(JSON.stringify({
                    Cmd: isPressed ? "press" : "release",
                    Data: key
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
