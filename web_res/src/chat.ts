window.onload = () => {
    let conn: WebSocket;
    let msg = document.getElementById("msg")!;
    let log = document.getElementById("log")!;

    function appendLog(item : HTMLDivElement) {
        let doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
        log.appendChild(item);
        if (doScroll) {
            log.scrollTop = log.scrollHeight - log.clientHeight;
        }
    }

    document.getElementById("form")!.onsubmit = () => {
        if (!conn) {
            return false;
        }
        if (!(<HTMLInputElement>msg).value) {
            return false;
        }
        conn.send((<HTMLInputElement>msg).value);
        (<HTMLInputElement>msg).value = "";
        return false;
    };

    if (window["WebSocket"]) {
        conn = new WebSocket("ws://" + document.location.host + "/ws");
        conn.onclose = () => {
            var item = document.createElement("div");
            item.innerHTML = "<b>Connection closed.</b>";
            appendLog(item);
        };
        conn.onmessage = (evt) => {
            var messages = evt.data.split('\n');
            for (var i = 0; i < messages.length; i++) {
                var item = document.createElement("div");
                item.innerText = messages[i];
                appendLog(item);
            }
        };
    } else {
        let item = document.createElement("div");
        item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
        appendLog(item);
    }
};