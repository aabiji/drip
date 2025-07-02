use std::sync::Arc;

use tokio::sync::mpsc;
use tokio::sync::Mutex;

use super::mdns::{Peer, Status, MDNS};

#[derive(Default)]
pub struct PeerToPeerService {
    pub peers: Vec<Peer>,
}

impl PeerToPeerService {
    pub fn new() -> Arc<Mutex<Self>> {
        Arc::new(Mutex::new(PeerToPeerService::default()))
    }

    pub async fn run_mdns(shared_self: Arc<Mutex<Self>>) {
        let (sender, mut receiver) = mpsc::channel::<Status>(32);

        tokio::spawn(async move {
            let mdns = MDNS::new(false);
            mdns.register_our_device();
            mdns.discover_peers(sender).await;
            mdns.shutdown();
        });

        tokio::spawn(async move {
            while let Some(message) = receiver.recv().await {
                let mut this = shared_self.lock().await;
                match message {
                    Status::PeerFound(peer) => {
                        this.peers.push(peer);
                        println!("message!");
                    }
                    Status::PeerLost { id } => this.peers.retain(|p| p.id != id),
                }
            }
        });
    }
}
