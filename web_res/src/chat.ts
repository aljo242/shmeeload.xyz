function openPopUpForm() {
    let style = document.getElementsByClassName("loginPopUp");
    if (style == null) {
        console.log("Unable to access login form!")
    }
    console.log(style);
    document.getElementById("popUpForm")!.style.display = "block";
}


function closePopUpForm() {
    let style = document.getElementsByClassName("loginPopUp");
    if (style == null) {
        console.log("Unable to access login form!")
    }
    console.log(style);
    document.getElementById("popUpForm")!.style.display = "none";

}

window.onclick = (event : MouseEvent) => {
    let modal = document.getElementById("popUpForm")!;
    console.log(modal)
    if (event.target != modal) {
        console.log(event)
        closePopUpForm()
    }
}

window.onload = () => {
    openPopUpForm()
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
            let item = document.createElement("div");
            item.innerHTML = "<b>Connection closed.</b>";
            appendLog(item);
        };
        conn.onmessage = (evt) => {
            let messages : string[] = evt.data.split('\n');
            for (let i = 0; i < messages.length; i++) {
                let item = document.createElement("div");
                item.innerText = "Alex: " + messages[i];
                appendLog(item);
            }
        };
    } else {
        let item = document.createElement("div");
        item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
        appendLog(item);
    }
};

