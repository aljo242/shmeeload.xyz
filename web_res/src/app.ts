const msg = "Hello web"

function printMessage(msg: string) {
    console.log(msg)
}

printMessage(msg)

const homeButton = document.getElementById("testHomeButton")!
homeButton.onclick = () => {
    printMessage("Button Pushed!")
}