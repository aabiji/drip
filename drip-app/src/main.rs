use dioxus::desktop::{tao::window::WindowBuilder, Config};
use dioxus::prelude::*;

use drip_net::p2p::{P2PService, SafeP2PService};
use drip_net::peer::ConnectionState;

use once_cell::sync::Lazy;

const MAIN_CSS: Asset = asset!("/assets/main.css");

const FILE_ICON: Asset = asset!("/assets/icons/file.png");
const UPLOAD_ICON: Asset = asset!("/assets/icons/upload.png");
const MOBILE_ICON: Asset = asset!("/assets/icons/smartphone.png");
const DESKTOP_ICON: Asset = asset!("/assets/icons/laptop.png");
const CHECKMARK_ICON: Asset = asset!("/assets/icons/checkmark.png");
const PLUS_ICON: Asset = asset!("/assets/icons/plus.png");

static SERVICE: Lazy<SafeP2PService> = Lazy::new(P2PService::safe_new);

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
        let clone = SERVICE.clone();
        spawn(async move {
            P2PService::run_mdns(clone).await;
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
    let peers = use_resource(move || async move { SERVICE.lock().await.peers.clone().keys() });

    let update_peer = move |index: usize| {
        let clone = SERVICE.clone();
        spawn(async move {
            P2PService::authorize_peer(clone, index).await;
        });
    };

    rsx! {
        div {
            h2 { "Devices" }
            match (*peers.read()).clone() {
                Some(peers) if peers.len() == 0 => rsx! { p { "No nearby devices found" } },
                None => rsx! { p { "Loading peers..." } },

                Some(peers) => rsx! {
                    for (index, peer) in peers.iter().enumerate() {
                        div {
                            class: "device",
                            img { src: if peer.is_mobile { MOBILE_ICON } else { DESKTOP_ICON } }
                            h3 { "{peer.id}" }

                            match peer.state {
                                ConnectionState::Connected => rsx! {
                                    button {
                                        class: "status connected",
                                        onclick: move |_| update_peer(index),
                                        img { src: CHECKMARK_ICON }
                                        "Connected"
                                    }
                                },
                                ConnectionState::Connecting => rsx! {
                                    button {
                                        class: "status connecting",
                                        img { src: CHECKMARK_ICON } // TODO: loading animation
                                        "Connected"
                                    }
                                },
                                ConnectionState::Disconnected => rsx! {
                                    button {
                                        class: "status disconnected",
                                        img { src: PLUS_ICON }
                                        "Add"
                                    }
                                },
                            }
                        }
                    }
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
