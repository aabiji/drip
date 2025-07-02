use std::net::IpAddr;

use mdns_sd::{ServiceDaemon, ServiceEvent, ServiceInfo};
use tokio::sync::mpsc::{self, Sender};

#[derive(Debug)]
enum PeerDiscovery {
    PeerFound { ip: IpAddr, id: String },
    PeerLost { id: String },
}

struct MDNS {
    daemon: ServiceDaemon,
    our_id: String,
    our_service_type: String,
    port: u16,
}

impl MDNS {
    fn new(debug_mode: bool) -> Self {
        let daemon = ServiceDaemon::new().expect("Failed to create mdns daemon");
        let (our_id, port) = if debug_mode {
            (
                // use command line flags for testing
                std::env::args().nth(1).unwrap(),
                std::env::args().nth(2).unwrap().parse::<u16>().unwrap(),
            )
        } else {
            (whoami::devicename(), 8081 as u16)
        };

        Self {
            daemon,
            our_id,
            our_service_type: String::from("_fileshare._tcp.local."),
            port,
        }
    }

    fn register_our_device(&self) {
        let our_ip = local_ip_address::local_ip().unwrap();
        let hostname = format!("{}.local.", our_ip);

        // Make the our id act as the instance name so we can
        // parse it from a service's fullname, which is:
        // {instance_name}.{service_type}
        let service = ServiceInfo::new(
            &self.our_service_type,
            &self.our_id,
            &hostname,
            our_ip,
            self.port,
            None,
        )
        .unwrap();

        self.daemon
            .register(service)
            .expect("Failed to register mdns service");
    }

    async fn discover_peers(&self, sender: Sender<PeerDiscovery>) {
        let receiver = self
            .daemon
            .browse(&self.our_service_type)
            .expect("Failed to browse mdns services");

        while let Ok(event) = receiver.recv() {
            match event {
                ServiceEvent::ServiceResolved(info) => {
                    let peer_ip = info.get_addresses().iter().next().unwrap();
                    let peer_id = info.get_fullname().split(".").next().unwrap();

                    if info.get_type() == self.our_service_type && peer_id != self.our_id {
                        sender
                            .send(PeerDiscovery::PeerFound {
                                ip: *peer_ip,
                                id: peer_id.to_string(),
                            })
                            .await
                            .unwrap();
                    }
                }

                ServiceEvent::ServiceRemoved(service_type, fullname) => {
                    if service_type == self.our_service_type {
                        let peer_id = fullname.split(".").next().unwrap();
                        sender
                            .send(PeerDiscovery::PeerLost {
                                id: peer_id.to_string(),
                            })
                            .await
                            .unwrap();
                    }
                }

                _ => {}
            }
        }
    }

    fn shutdown(&self) {
        self.daemon.shutdown().unwrap();
    }
}

#[tokio::main]
async fn main() {
    let (sender, mut receiver) = mpsc::channel::<PeerDiscovery>(32);

    let handle = tokio::spawn(async move {
        let mdns = MDNS::new(true);
        mdns.register_our_device();
        mdns.discover_peers(sender).await;
        mdns.shutdown();
    });

    // TODO: now that we have peer discovery messages passed to a channel
    // we can now work on initializing a WebRTC connection over TCP
    while let Some(message) = receiver.recv().await {
        println!("Message: {:?}", message);
    }

    handle.await.unwrap();
}
