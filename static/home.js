// Copyright 2012 Lawrence Kesteloot

(function () {
    var SHOW_DEBUG = false;
    var g_ws = null;

    // Set up the DOM for the screen, which is an array of spans of fixed size with the
    // same background (font.png). We move the background around for each cell to show
    // a different character. This works well but makes it impossible to copy and
    // paste the text.
    var createScreen = function () {
        var $screen = $("div.screen");

        var addr = 15360;
        for (var y = 0; y < 16; y++) {
            for (var x = 0; x < 64; x++) {
                var $ch = $("<span>").attr("id", "s" + addr).addClass("char");
                if (x % 2 === 0) {
                    $ch.addClass("even-column");
                } else {
                    $ch.addClass("odd-column");
                }
                $screen.append($ch);
                addr++;
            }
            $screen.append($("<br>"));
        }
    };

    // Create the action buttons, the various messages, motor lights, and other
    // controls.
    var createControlPanel = function () {
        var $controlPanel = $("td.control-panel");

        $("#bootButton").click(function () {
            if (g_ws) {
                g_ws.send(JSON.stringify({Cmd: "boot"}));
            }
        });
        $("#resetButton").click(function () {
            if (g_ws) {
                g_ws.send(JSON.stringify({Cmd: "reset"}));
            }
        });

        if (SHOW_DEBUG) {
            $(".debug-panel").show();
        }

        $("#traceButton").click(function () {
            if (g_ws) {
                g_ws.send(JSON.stringify({Cmd: "tron"}));
            }
        });
        $("#addBreakpointButton").click(function () {
            if (g_ws) {
                var $breakpointAddress = $("#breakpointAddress");
                var addr = parseInt($breakpointAddress.val(), 16);
                g_ws.send(JSON.stringify({Cmd: "add_breakpoint", Addr: addr}));
                $breakpointAddress.val("");
                $("#message").text("Breakpoint set at 0x" + addr.toString(16))
            }
        });

        // Configure the control where the user can specify diskettes and cassette.
        var configureInputSelector = function (input, file_type) {
            var $select = $("#" + input);

            // Fill the <select>
            $.ajax({
                url: "/" + file_type + ".json",
                dataType: "json",
                success: function (filenames) {
                    $select.empty();
                    $select.append(
                        $("<option>").
                            text("-- empty --"));
                    for (var i = 0; i < filenames.length; i++) {
                        $select.append(
                            $("<option>").
                                text(filenames[i]));
                    }
                },
                error: function () {
                    $select.empty();
                    $select.append(
                        $("<option>").
                            text("-- invalid directory --"));
                }
            });

            // Update VM when input changes.
            var setInput = function () {
                var filename = $select.find("option:selected").text();
                if (filename.charAt(0) === "-") {
                    filename = "";
                }

                if (g_ws) {
                    g_ws.send(JSON.stringify({Cmd: "set_" + input, Data: filename}));
                }

                // Blur the selector so that subsequent keystrokes (for the emulator)
                // don't change the selection.
                $select.blur();
            };

            // Look for changes.
            $select.change(setInput);
        };

        configureInputSelector("disk0", "disks");
        configureInputSelector("disk1", "disks");
        configureInputSelector("cassette", "cassettes");
    };

    // Handle a command from the emulator.
    var handleUpdate = function (update) {
        var cmd = update.Cmd;

        if (cmd === "poke") {
            // Poke a string at the address.
            var addr = update.Addr;

            for (var i = 0; i < update.Msg.length; i++) {
                var data = update.Msg.charCodeAt(i);

                if (addr >= 15360 && addr < 16384) {
                    // Screen.
                    var $s = $("#s" + addr);
                    var cls = $s.attr("class");
                    var newCls = "char char-" + data;

                    // Retain the odd/even columns. Could recompute this from
                    // the address too.
                    if (cls.indexOf("odd-column") >= 0) {
                        newCls += " odd-column";
                    } else {
                        newCls += " even-column";
                    }
                    $s.attr("class", newCls);
                }

                addr++;
            }
        } else if (cmd === "motor") {
            // Turn the diskette motor light on or off.
            var motorOn = update.Data != 0;
            var $motor;
            if (update.Addr == -1) {
                $motor = $("#motorCassette");
            } else {
                $motor = $("#motorDrive" + update.Addr);
            }
            if (motorOn) {
                $motor.addClass("motorLightOn");
            } else {
                $motor.removeClass("motorLightOn");
            }
        } else if (cmd === "breakpoint") {
            // We've hit a breakpoint. This could just be a message.
            $("#message").text("Breakpoint at 0x" + update.Addr.toString(16))
        } else if (cmd === "message") {
            // Show a generic message.
            $("#message").text(update.Msg);
        } else if (cmd === "expanded") {
            // Expanded character font.
            if (update.Data !== 0) {
                $("div.screen").addClass("expanded").removeClass("narrow");
            } else {
                $("div.screen").removeClass("expanded").addClass("narrow");
            }
        } else {
            console.log("Unknown command \"" + cmd + "\"");
        }
    };

    // Set up the web socket to get updates and send commands.
    var configureWs = function () {
        var ws = new WebSocket("ws://" + window.location.host + "/ws");
        ws.onmessage = function (event) {
            var updates = JSON.parse(event.data);
            for (var i = 0; i < updates.length; i++) {
                handleUpdate(updates[i]);
            }
        };
        ws.onclose = function (event) {
            g_ws = null;
        };
        return ws;
    };

    // Convert keys on the keyboard to ASCII letters or special strings like "Enter".
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
            } else if (which == 9) {
                // Tab.
                key = "Clear";

                // Don't move focus to next field.
                event.preventDefault();
            } else {
                // Ignore.
                /// console.log(which);
                key = "";
            }

            return key
        };

        // Handle a key event by mapping it and sending it to the emulator.
        var keyEvent = function (event, isPressed) {
            // Don't send to virtual computer if a text input field is selected.
            if ($(document.activeElement).attr("type") == "text") {
                return;
            }

            var key = eventToKey(event);
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
        createControlPanel();
        g_ws = configureWs();
        configureKeyboard();
    });
})();
