pub mod mdns;
pub mod peer;

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::Mutex;
use tokio::sync::mpsc;

// TODO: fix main.rs

#[derive(Default)]
pub struct P2PService {
    pub peers: HashMap<String, peer::Peer>,
}

pub type SafeP2PService = Arc<Mutex<P2PService>>;

impl P2PService {
    pub fn safe_new() -> SafeP2PService {
        Arc::new(Mutex::new(P2PService::default()))
    }

    // TODO: use tokio signals to stop these on background process close
    pub async fn run_mdns(shared_self: Arc<Mutex<Self>>) {
        let (sender, mut receiver) = mpsc::channel::<mdns::Status>(32);

        tokio::spawn(async move {
            let muticast = Arc::new(mdns::MDNS::new(true)); // TODO: change this in production
            muticast.register_our_device();
            muticast.discover_peers(sender).await;
            muticast.shutdown();
        });

        tokio::spawn(async move {
            while let Some(message) = receiver.recv().await {
                let mut this = shared_self.lock().await;
                match message {
                    mdns::Status::PeerFound(info) => {
                        let id = info.id.clone();
                        let peer = peer::Peer::new(info).await;
                        this.peers.insert(id, peer);
                    }
                    mdns::Status::PeerLost { id } => { this.peers.remove(&id).unwrap(); },
                }
            }
        });
    }

    pub async fn connect_peer(shared_self: Arc<Mutex<Self>>, index: usize) {
        todo!();
    }

    pub async fn disconnect_peer(shared_self: Arc<Mutex<Self>>, index: usize) {
        todo!();
    }
}
