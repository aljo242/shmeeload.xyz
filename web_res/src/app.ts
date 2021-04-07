import * as colors from "./color_schemes";

console.log(colors.scheme1)

// Register Service Worker
if ("serviceWorker" in navigator) {
    window.addEventListener("load", function() {
      navigator.serviceWorker.register("/serviceWorker.js").then(function(registration) {
        // Registration was successful
        console.log("ServiceWorker registration successful with scope: ", registration.scope);
      }, function(err) {
        // registration failed :(
        console.log("ServiceWorker registration failed: ", err);
      });
    });
  }


const msg = "Hello web"

const homeButton = document.getElementById("secretButton")!
homeButton.onclick = () => {
    printMessage("Button Pushed!");
};



function printMessage(msg: string) {
    console.log(msg)
}


printMessage(msg)
