let signInName : string = "anon";
const DEFAULT_NAME : string = "anon";

function convertArrayBufferToNumber(buffer: ArrayBuffer) {
    const bytes = new Uint8Array(buffer);
    const dv = new DataView(bytes.buffer);
    return dv.getUint16(0, true);
}

class User {
    userName: string;
    conn: WebSocket;

    constructor(name: string, conn: WebSocket) {
        console.log("Creating new User...")
        console.log(this)
        this.userName = name;
        this.conn = conn;

        this.signIn();
    }

    signIn() {
        const signInMessage = this.userName + " signed in!";
        this.conn.send(signInMessage);
        console.log(signInMessage);

        let item = document.createElement("div");
        item.innerHTML = "<b>Connection closed.</b>";
        appendLog(item);
    }
}

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

//Since you know what type you are expecting, but the type 
//system can't, to get this to work, you have to tell Typescript 
//what type of element you expect to be selecting. You would do that 
//through casting the type of the selected element as follows: 
//const inputElement = <HTMLInputElement> document.getElementById("food-name-val"); 
//or 
//const inputElement = document.getElementById("food-name-val") as HTMLInputElement;


window.onclick = (event : MouseEvent) => {
    let modal = document.getElementById("popUpForm")!;
    if (event.target == modal) {
        closePopUpForm();
    }
}

function appendLog(item : HTMLDivElement) {
    let log = document.getElementById("log")!;
    const doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
    log.appendChild(item);
    if (doScroll) {
        log.scrollTop = log.scrollHeight - log.clientHeight;
    }
    console.log(log);

}


window.onload = () => {
    openPopUpForm()
    let conn: WebSocket;
    let user: User;
    let msg = document.getElementById("msg")! as HTMLInputElement;

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
        return
    }

    let signInButton = document.getElementById("signInButton")!;
    signInButton.onclick = () => {
        let userName = document.getElementById("chatname") as HTMLInputElement;
        if (userName.value != "") {
            user = new User(userName.value, conn);
            closePopUpForm(); 
        } else {
            console.log("USER INPUT ERROR");
            user = new User(DEFAULT_NAME, conn);
            closePopUpForm(); 
        }
    }

    let msgForm = document.getElementById("signInButton")!;
    msgForm.onsubmit = () => {
        if (!conn) {
            return false;
        }
        if (!msg.value) {
            return false;
        }
        const messageWithName = user.userName + ": " + msg.value;
        conn.send(messageWithName);
        msg.value = "";
        return false;
    };
};

