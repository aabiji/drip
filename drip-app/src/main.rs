use dioxus::desktop::{tao::window::WindowBuilder, Config};
use dioxus::prelude::*;

use drip_net::p2p::{P2PService, SafeP2PService};

const MAIN_CSS: Asset = asset!("/assets/main.css");

const FILE_ICON: Asset = asset!("/assets/icons/file.png");
const UPLOAD_ICON: Asset = asset!("/assets/icons/upload.png");
const MOBILE_ICON: Asset = asset!("/assets/icons/smartphone.png");
const DESKTOP_ICON: Asset = asset!("/assets/icons/laptop.png");
const CHECKMARK_ICON: Asset = asset!("/assets/icons/checkmark.png");
const PLUS_ICON: Asset = asset!("/assets/icons/plus.png");

static SERVICE: GlobalSignal<SafeP2PService> = Global::new(|| P2PService::new());

fn main() {
    dioxus::LaunchBuilder::new()
        .with_cfg(
            Config::default()
                .with_menu(None)
                .with_window(WindowBuilder::new().with_maximized(true).with_title("Drip")),
        )
        .launch(App);
}

#[component]
fn App() -> Element {
    use_effect(move || {
        let service = SERVICE.read().clone();

        spawn(async move {
            P2PService::run_mdns(service).await;
        });
    });

    rsx! {
        document::Link { rel: "stylesheet", href: MAIN_CSS }
        Home {}
    }
}

#[component]
fn Home() -> Element {
    let mut showing_files = use_signal(|| true);

    rsx! {
        div {
            class: "navbar",
            button {
                class: if *showing_files.read() { "active" } else { "" },
                onclick: move |_| showing_files.set(true),
                "Files"
            }
            button {
                class: if !*showing_files.read() { "active" } else { "" },
                onclick: move |_| showing_files.set(false),
                "Settings"
            }
        }

        if *showing_files.read() {
             FileView{}
        } else {
             SettingsView{}
        }
    }
}

#[component]
fn FileView() -> Element {
    let example_files = [
        "File A", "File B", "File C", "File D", "File E", "File F", "File G", "File H", "File J",
    ];

    rsx! {
        label {
            class: "file-picker",
            "Drag and drop or select files"
                input {
                    r#type: "file",
                    multiple: true,
                    onchange: move |event| {
                        // handle files here
                    },
                    img { src: UPLOAD_ICON }
                }
        }

        div {
            class: "file-grid",

            for file in example_files {
                button {
                    class: "file",
                    onclick: move |event| {
                        // copy file path to clipboard
                    },

                    div {
                        img { src: FILE_ICON }
                        div {
                            class: "info",
                            p { "{file}" },
                            p { "Size" },
                            p { "Sent from" }
                        }
                    }
                }
            }
        }
    }
}

#[component]
fn DeviceList() -> Element {
    // TODO: this is buggy
    let peers = use_resource(move || async move {
        let service = SERVICE.read().clone();
        let guard = service.lock().await;
        guard.peers.clone()
    });

    rsx! {
        div {
            h2 { "Devices" }
            match (*peers.read()).clone() {
                Some(peers) => rsx! {
                    for peer in peers.iter() {
                        div {
                            class: "device",
                            img { src: if peer.is_mobile { MOBILE_ICON } else { DESKTOP_ICON } }
                            h3 { "{peer.id}" }
                            button {
                                class: "status connected",
                                img { src: CHECKMARK_ICON }
                                "Connected"
                            }
                        }
                    }
                },
                None => rsx! {
                    p { "Loading peers..." }
                }
            }
        }
    }
}

#[component]
fn SettingsView() -> Element {
    rsx! {
        div {
            class: "settings-container",

            DeviceList {}
            hr {}

            div {
                h2 { "Misc" }

                label {
                    class: "destination-picker",
                    "Download path"
                        input {
                            r#type: "file",
                            multiple: true,
                            onchange: move |event| {
                                // handle path here
                            },
                        }
                }

                button {
                    "Toggle theme"
                }
            }

            p {
                class: "footer",
                "(C) Abigail Adegbiji, 2025 - now"
            }
        }
    }
}
