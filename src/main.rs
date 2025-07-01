/*
use dioxus::prelude::*;

mod backend;

#[derive(Debug, Clone, Routable, PartialEq)]
#[rustfmt::skip]
enum Route {
    #[route("/")]
    Home {},
}

const MAIN_CSS: Asset = asset!("/assets/main.css");

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    rsx! {
        document::Link { rel: "stylesheet", href: MAIN_CSS }
        Router::<Route> {}
    }
}

#[component]
fn Home() -> Element {
    rsx! {
        h1 { "Your files" }
        button { onclick: |_event| backend::demo(), "hello!" }
    }
}
*/
use std::sync::Arc;

mod net;
use net::discovery;

fn main() {
    // Background process (TODO: should be an actual process)
    // that handles peer discovery
    let peers = discovery::new_peers();

    // TODO (for both threads): should we restart???
    // What errors can the background process recover from?
    // At what point do we tell the user?
    let clone1 = Arc::clone(&peers);
    let handler1 = std::thread::spawn(move || {
        if let Err(err) = discovery::run_client(clone1) {
            panic!("{}", err);
        }
    });

    let clone2 = Arc::clone(&peers);
    let handler2 = std::thread::spawn(move || {
        if let Err(err) = discovery::run_server(clone2) {
            panic!("{}", err);
        }
    });

    let _ = handler1.join();
    let _ = handler2.join();
}