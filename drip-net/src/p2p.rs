use std::io::BufReader;
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;

use tokio::sync::{mpsc, Mutex};

use super::mdns::{Status, MDNS};
use super::peer::Peer;

#[derive(Default)]
pub struct P2PService {
    pub peers: Vec<Peer>,
}

pub type SafeP2PService = Arc<Mutex<P2PService>>;

struct Packet {
    peer_id: String,
    data_type: String,
    data: String,
}

// TODO: implement the webrtc state machine...
fn start_tcp_client() {
    let addr = format!("{peer.id}:{peer.port}");
    let mut stream = TcpStream::connect(addr).unwrap();

    let answer_packet = Packet { peer_id, data_type: "sdp_offer", answer }; 
    let offer_packet = Packet { peer_id, data_type: "sdp_offer", offer }; 
    let candidate_packet = Packet { peer_id, data_type: "ice", candidate }; 
}

fn start_tcp_server() {
    let addr = format!("{our_id}:{our_port}");
    let listener = TcpListener::bind(addr).unwrap();

    for stream in listener.incoming() {
        let mut buffered = BufReader::new(stream.unwrap());
        let packet = serde_json::from_reader(buffered).unwrap();

        let polite = our_id.to_lowercase() < packet.peer_id.to_lowercase();

        match packet.data_type {
            "sdp_offer" => todo!("handle sdp offer"),
            "sdp_answer" => todo!("handle sdp answer"),
            "ice" => todo!("handle ice candidate"),
            _ => todo!("unknown packet type");
        }
    }
}

impl P2PService {
    pub fn new() -> SafeP2PService {
        Arc::new(Mutex::new(P2PService::default()))
    }

    // TODO: use tokio signals to stop these on background process close
    pub async fn run_mdns(shared_self: Arc<Mutex<Self>>) {
        let (sender, mut receiver) = mpsc::channel::<Status>(32);

        tokio::spawn(async move {
            let mdns = MDNS::new(true); // TODO: change this in production
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

    pub async fn connect_peer(shared_self: Arc<Mutex<Self>>, index: usize) {
        // webrtc connect
    }

    pub async fn disconnect_peer(shared_self: Arc<Mutex<Self>>, index: usize) {}
}
