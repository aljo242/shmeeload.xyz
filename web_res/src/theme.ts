import { getElement } from "./dom.js";

// Shared color-scheme theming, used by both the main pages and the chat. The
// secret shmee-man button cycles to a new random scheme.

type Color = string;
const black: Color = "#000000";

class ColorScheme {
    article_background: Color = black;
    article_opacity = "1.0";
    nav_background: Color = black;
    nav_header: Color = black;
    nav_opacity = "1.0";
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

const scheme1 = new ColorScheme();
scheme1.article_background = "#f4a261";
scheme1.nav_background = "#264653";
scheme1.nav_header = "#e76f51";
scheme1.body_background = black;
scheme1.hr = black;
scheme1.a_nav_link = "#14ff14";
scheme1.a_text_link = "#e9c46a";

const scheme2 = new ColorScheme();
scheme2.article_background = "#f0e68c";
scheme2.nav_background = "#adff2f";
scheme2.nav_header = "#ffc0cb";
scheme2.body_background = black;
scheme2.hr = black;
scheme2.a_nav_link = "#f700ff";
scheme2.a_text_link = "#f700ff";

const scheme3 = new ColorScheme();
scheme3.article_background = "#e5e5e5";
scheme3.nav_background = "#e5e5e5";
scheme3.nav_header = "#fb8500";
scheme3.body_background = black;
scheme3.hr = black;
scheme3.a_nav_link = "#52b788";
scheme3.a_text_link = "#52b788";

const scheme4 = new ColorScheme();
scheme4.article_background = "#e2fdff";
scheme4.nav_background = "#bfd7ff";
scheme4.nav_header = "#5465ff";
scheme4.body_background = black;
scheme4.hr = black;
scheme4.a_nav_link = "#c32f27";
scheme4.a_text_link = "#c32f27";

const scheme5 = new ColorScheme();
scheme5.article_background = "#f28482";
scheme5.article_opacity = ".975";
scheme5.nav_background = "#84a59d";
scheme5.nav_header = "#f7ede2";
scheme5.nav_opacity = ".975";
scheme5.body_background = black;
scheme5.hr = black;
scheme5.a_nav_link = "#14ff14";
scheme5.a_text_link = "#14ff14";

const scheme6 = new ColorScheme();
scheme6.article_background = "#ddc9b4";
scheme6.article_opacity = ".95";
scheme6.nav_background = "#bcac9b";
scheme6.nav_header = "#c17c74";
scheme6.nav_opacity = ".90";
scheme6.body_background = black;
scheme6.hr = black;
scheme6.a_nav_link = "#df2935";
scheme6.a_text_link = "#df2935";

const scheme7 = new ColorScheme();
scheme7.article_background = "#dddf00";
scheme7.article_opacity = ".95";
scheme7.nav_background = "#2b9348";
scheme7.nav_header = "#007f5f";
scheme7.nav_opacity = ".90";
scheme7.body_background = black;
scheme7.hr = black;
scheme7.a_nav_link = "#001d3d";
scheme7.a_text_link = "#001d3d";

function applyColorScheme(scheme: ColorScheme) {
    getElement("header").style.backgroundColor = scheme.nav_header;

    const nav = getElement("nav");
    nav.style.backgroundColor = scheme.nav_background;
    nav.style.opacity = scheme.nav_opacity;

    const mainArticleBody = getElement("mainArticleBody");
    mainArticleBody.style.backgroundColor = scheme.article_background;
    mainArticleBody.style.opacity = scheme.article_opacity;

    for (const link of document.getElementsByTagName("a")) {
        link.style.color = scheme.a_nav_link;
    }
}

const colorSchemes = [scheme1, scheme2, scheme3, scheme4, scheme5, scheme6, scheme7];
let currentSelection = 0;
let prevSelection = 0;

function applyRandomColorScheme() {
    let random = Math.floor(Math.random() * colorSchemes.length);
    // make sure we always choose a unique one
    while (random === currentSelection || random === prevSelection) {
        random = Math.floor(Math.random() * colorSchemes.length);
    }
    prevSelection = currentSelection;
    currentSelection = random;
    applyColorScheme(colorSchemes[currentSelection]);
}

// initTheme applies the default scheme and wires the secret button to cycle.
export function initTheme() {
    applyColorScheme(colorSchemes[currentSelection]);
    getElement("secretButton").addEventListener("click", applyRandomColorScheme);
}
