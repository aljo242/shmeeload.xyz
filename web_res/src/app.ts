// type alias
type Color = string;
const black: Color = "#000000"

class ColorScheme {
    article_background: Color = black;
    nav_background: Color = black;
    nav_header: Color = black;
    body_background: Color = black;
    hr: Color = black;

    a_nav_link: Color = black;
    a_nav_visited: Color = black;
    a_nav_hover: Color = black;
    a_nav_active: Color = black;

    a_text_link: Color = black;
    a_text_visited: Color = black;
    a_text_hover: Color = black;
    a_text_active: Color = black;

    main_text: Color = black;
}

let scheme1 = new ColorScheme;
scheme1.article_background = "#f4a261";
scheme1.nav_background = "#264653";
scheme1.nav_header = "#e76f51";
scheme1.body_background = black;
scheme1.hr = black;
scheme1.a_nav_link = "#14ff14";
scheme1.a_text_link = "#e9c46a";



let scheme2 = new ColorScheme;
scheme2.article_background = "#f0e68c";
scheme2.nav_background = "#adff2f";
scheme2.nav_header = "#ffc0cb";
scheme2.body_background = black;
scheme2.hr = black;
scheme2.a_nav_link = "#f700ff";
scheme2.a_text_link = "#f700ff";


let scheme3 = new ColorScheme;
scheme3.article_background = "#e5e5e5";
scheme3.nav_background = "#e5e5e5";
scheme3.nav_header = "#fb8500";
scheme3.body_background = black;
scheme3.hr = black;
scheme3.a_nav_link = "#52b788";

scheme3.a_text_link = "#52b788";


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
  //console.log(scheme)
  let header = document.getElementById("header")!;
  header.style.backgroundColor = scheme.nav_header;
  let nav = document.getElementById("nav")!;
  nav.style.backgroundColor = scheme.nav_background;
  let mainArticleBody = document.getElementById("mainArticleBody")!;
  mainArticleBody.style.backgroundColor = scheme.article_background
  
  //let navLinks = document.getElementsByClassName("navLink")!;
  //for (let item  of navLinks) {
  //  let hmtlItem = <HTMLElement>item;
  //  hmtlItem.style.color = scheme.a_nav_link;
  //}

  //let textLinks = document.getElementsByClassName("textLink")!;
  //for (let item  of textLinks) {
  //  let hmtlItem = <HTMLElement>item;
  //  hmtlItem.style.color = scheme.a_text_link;
  //}

  let links = document.getElementsByTagName("a")!;
  for (let item  of links) {
    item.style.color = scheme.a_nav_link;
  }
}

const colorSchemes = [scheme1, scheme2, scheme3];
let currentSelection = 0;


function applyRandomColorScheme() {
  let random = Math.floor(Math.random() * colorSchemes.length);
  // make sure we always choose a unique one
  while (random === currentSelection) {
    random = Math.floor(Math.random() * colorSchemes.length);
  }
  currentSelection = random
  console.log(currentSelection)
  applyColorScheme(colorSchemes[currentSelection]);
}

applyColorScheme(colorSchemes[currentSelection]);

const homeButton = document.getElementById("secretButton")!
homeButton.onclick = () => {
    applyRandomColorScheme();
};


