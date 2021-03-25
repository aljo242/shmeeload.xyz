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

let deferredPrompt: Event;

const homeButton = document.getElementById("testHomeButton")!
homeButton.onclick = () => {
    printMessage("Button Pushed!");
};

window.addEventListener('beforeinstallprompt', (e: Event) => {
    // Prevent Chrome 67 and earlier from automatically showing the prompt
    e.preventDefault();
    // Stash the event so it can be triggered later.
    deferredPrompt = e;
    // Update UI notify the user they can add to home screen
    homeButton.style.display = 'block';
});


function printMessage(msg: string) {
    console.log(msg)
}

printMessage(msg)

