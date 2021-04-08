// type alias
type Color = string;
const black: Color = "#000000"

class ColorScheme {
    article_background: Color = black;
    nav_background: Color = black;
    nav_header: Color = black;
    body_background: Color = black;
    hr: Color = black;

    a_link: Color = black;
    a_visited: Color = black;
    a_hover: Color = black;
    a_active: Color = black;

    main_text: Color = black;
}

let scheme1 = new ColorScheme;
scheme1.article_background = "#f4a261";
scheme1.nav_background = "#264653";
scheme1.nav_header = "#e76f51";
scheme1.body_background = black;
scheme1.hr = black;
scheme1.a_link = "#e9c46a";
scheme1.a_visited = "e9c46a";
scheme1.a_hover = "e9c46a";
scheme1.a_active = "e9c46a";

let scheme2 = new ColorScheme;
scheme2.article_background = "#f0e68c";
scheme2.nav_background = "#adff2f";
scheme2.nav_header = "#ffc0cb";
scheme2.body_background = black;
scheme2.hr = black;
scheme2.a_link = "#f700ff";
scheme2.a_visited = "f700ff";
scheme2.a_hover = "ffd038";
scheme2.a_active = "ffff00";

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





function printMessage(msg: string) {
    console.log(msg)
}


function applyColorScheme(scheme: ColorScheme) {
  console.log(scheme)
  let header = document.getElementById("header")!;
  header.style.backgroundColor = scheme.nav_header;
  let nav = document.getElementById("nav")!;
  nav.style.backgroundColor = scheme.nav_background;
  let mainArticleBody = document.getElementById("mainArticleBody")!;
  mainArticleBody.style.backgroundColor = scheme.article_background
  let navLinks = document.getElementsByTagName("a")!;
  for (let item  of navLinks) {
    console.log(item.style)
    item.style.color = scheme.a_link;
  }
}

const colorSchemes = [scheme1, scheme2];


function applyRandomColorScheme() {
  const random = Math.floor(Math.random() * colorSchemes.length);
  applyColorScheme(colorSchemes[random]);
}


const homeButton = document.getElementById("secretButton")!
homeButton.onclick = () => {
    printMessage("Button Pushed!");
    applyRandomColorScheme();
};


