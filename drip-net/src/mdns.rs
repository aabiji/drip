use mdns_sd::{ServiceDaemon, ServiceEvent, ServiceInfo};

use tokio::sync::mpsc::Sender;

use super::peer::PeerInfo;

pub enum Status {
    PeerFound(PeerInfo),
    PeerLost { id: String },
}

pub struct MDNS {
    daemon: ServiceDaemon,
    our_id: String,
    our_service_type: String,
}

impl MDNS {
    pub fn new(debug_mode: bool) -> Self {
        let daemon = ServiceDaemon::new().expect("Failed to create mdns daemon");
        let our_id = if debug_mode {
            format!("peer-{}", rand::random_range(0..10))
        } else {
            whoami::devicename()
        };

        Self {
            daemon,
            our_id,
            our_service_type: String::from("_fileshare._tcp.local."),
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
            8081,
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
                    let ip = info.get_addresses().iter().next().unwrap();
                    let id = info.get_fullname().split(".").next().unwrap();
                    let mobile = info.get_property_val_str("is_mobile").unwrap() == "true";

                    if info.get_type() == self.our_service_type && id != self.our_id {
                        let polite = id.to_lowercase() < self.our_id.to_lowercase();

                        sender
                            .send(Status::PeerFound(PeerInfo{
                                ip: *ip,
                                our_id: self.our_id.clone(),
                                id: id.to_string(),
                                mobile,
                                polite
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
