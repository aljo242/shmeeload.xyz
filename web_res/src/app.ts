const msg = "Hello web"

function printMessage(msg: string) {
    console.log(msg)
}

printMessage(msg)

const button = document.getElementById("testButton")!
button.onclick = () => {
    printMessage("Button Pushed!")
}