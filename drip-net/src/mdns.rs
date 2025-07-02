use std::net::IpAddr;

use mdns_sd::{ServiceDaemon, ServiceEvent, ServiceInfo};
use tokio::sync::mpsc::Sender;

#[derive(Debug, Clone)]
pub struct Peer {
    pub ip: String,
    pub id: String,
    pub is_mobile: bool,
}

#[derive(Debug)]
pub enum Status {
    PeerFound(Peer),
    PeerLost { id: String },
}

pub struct MDNS {
    daemon: ServiceDaemon,
    our_id: String,
    our_service_type: String,
    port: u16,
}

impl MDNS {
    pub fn new(debug_mode: bool) -> Self {
        let daemon = ServiceDaemon::new().expect("Failed to create mdns daemon");
        let (our_id, port) = if debug_mode {
            (
                // use command line flags for testing
                std::env::args().nth(1).unwrap(),
                std::env::args().nth(2).unwrap().parse::<u16>().unwrap(),
            )
        } else {
            (whoami::devicename(), 8081_u16)
        };

        Self {
            daemon,
            our_id,
            our_service_type: String::from("_fileshare._tcp.local."),
            port,
        }
    }

    pub fn register_our_device(&self) {
        let our_ip = local_ip_address::local_ip().unwrap();
        let hostname = format!("{our_ip}.local.");

        let is_mobile = std::env::consts::OS == "android" || std::env::consts::OS == "ios";
        let properties = [("is_mobile", is_mobile)];

        // Make the our id act as the instance name so we can
        // parse it from a service's fullname, which is:
        // {instance_name}.{service_type}
        let service = ServiceInfo::new(
            &self.our_service_type,
            &self.our_id,
            &hostname,
            our_ip,
            self.port,
            &properties[..],
        )
        .unwrap();

        self.daemon
            .register(service)
            .expect("Failed to register mdns service");
    }

    pub async fn discover_peers(&self, sender: Sender<Status>) {
        let receiver = self
            .daemon
            .browse(&self.our_service_type)
            .expect("Failed to browse mdns services");

        while let Ok(event) = receiver.recv_async().await {
            match event {
                ServiceEvent::ServiceResolved(info) => {
                    let peer_ip = info.get_addresses().iter().next().unwrap();
                    let peer_id = info.get_fullname().split(".").next().unwrap();
                    let is_mobile = info.get_property_val_str("is_mobile").unwrap() == "true";

                    if info.get_type() == self.our_service_type && peer_id != self.our_id {
                        sender
                            .send(Status::PeerFound(Peer {
                                ip: *peer_ip,
                                id: peer_id.to_string(),
                                is_mobile,
                            }))
                            .await
                            .unwrap();
                    }
                }

                ServiceEvent::ServiceRemoved(service_type, fullname) => {
                    if service_type == self.our_service_type {
                        let peer_id = fullname.split(".").next().unwrap();
                        sender
                            .send(Status::PeerLost {
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

    pub fn shutdown(&self) {
        self.daemon.shutdown().unwrap();
    }
}
