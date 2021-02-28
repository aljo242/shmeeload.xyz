let signInName : string = "anon";

function openPopUpForm() {
    let form = document.getElementById("popUpForm");
    if (form == null) {
        console.log("Unable to access login form!");
        return;
    }  
    form.style.display = "block";
}

function closePopUpForm() {
    let form = document.getElementById("popUpForm");
    if (form == null) {
        console.log("Unable to access login form!");
        return;
    }  
    form.style.display = "none";
}

function signIn() {
    let userName = document.getElementById("chatname")! as HTMLInputElement;
    signInName = userName.value;
    console.log(signInName + " signed in!");
    closePopUpForm();
}

//Since you know what type you are expecting, but the type 
//system can't, to get this to work, you have to tell Typescript 
//what type of element you expect to be selecting. You would do that 
//through casting the type of the selected element as follows: 
//const inputElement = <HTMLInputElement> document.getElementById("food-name-val"); 
//or 
//const inputElement = document.getElementById("food-name-val") as HTMLInputElement;



function signInDefault() {
    console.log(signInName + " signed in!")
    closePopUpForm();
}

window.onclick = (event : MouseEvent) => {
    let modal = document.getElementById("popUpForm")!;
    if (event.target == modal) {
        closePopUpForm();
    }
}

window.onload = () => {
    openPopUpForm()
    let conn: WebSocket;
    let msg = document.getElementById("msg")! as HTMLInputElement;
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
        if (!msg.value) {
            return false;
        }
        let messageWithName = signInName + ": " + msg.value;
        conn.send(messageWithName);
        msg.value = "";
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

