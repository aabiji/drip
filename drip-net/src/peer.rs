use std::net::{IpAddr, SocketAddr};
use std::sync::Arc;
use std::boxed::Box;

use tokio::sync::Mutex;
use tokio::sync::mpsc;
use tokio::sync::mpsc::{Sender, Receiver};
use tokio::net::UdpSocket;

use webrtc::peer_connection::RTCPeerConnection;
use webrtc::api::interceptor_registry::register_default_interceptors;
use webrtc::api::media_engine::MediaEngine;
use webrtc::api::APIBuilder;
use webrtc::error::Result;
use webrtc::interceptor::registry::Registry;
use webrtc::peer_connection::configuration::RTCConfiguration;
use webrtc::ice_transport::ice_candidate::RTCIceCandidate;
use webrtc::peer_connection::peer_connection_state::RTCPeerConnectionState;
use webrtc::peer_connection::sdp::session_description::RTCSessionDescription;
use webrtc::peer_connection::signaling_state::RTCSignalingState;
use webrtc::peer_connection::sdp::sdp_type::RTCSdpType;

#[derive(Clone)]
pub enum ConnectionState {
    Disconnected,
    Connecting,
    Connected,
}

#[derive(Clone)]
#[serde(tag = "type", content = "content")]
enum Data {
    Offer(String),
    Answer(String),
    ICE(String),
}

#[derive(Clone)]
struct Packet {
    data: Data,
    from: String,
    to: String,
}

#[derive(Clone)]
pub struct PeerInfo {
    pub id: String,
    pub our_id: String,
    pub ip: IpAddr,
    pub mobile: bool,
    pub polite: bool,
}

pub struct Peer {
    pub info: PeerInfo,
    pub state: Arc<Mutex<ConnectionState>>,

    sender: Sender<Packet>,
    receiver: Receiver<Packet>,

    making_offer: Arc<Mutex<bool>>,
    connection: Arc<RTCPeerConnection>,
}

async fn create_peer_connection() -> Result<RTCPeerConnection> {
    let mut m = MediaEngine::default();
    m.register_default_codecs()?;

    let mut registry = Registry::new();
    registry = register_default_interceptors(registry, &mut m)?;

    let api = APIBuilder::new()
        .with_media_engine(m)
        .with_interceptor_registry(registry)
        .build();

    let config = RTCConfiguration {
        ice_servers: vec![],
        ..Default::default()
    };

    Ok(api.new_peer_connection(config).await?)
}

// TODO: the impolite one should create a data channel and do data_channel.on_open
//       the polite one should call on_data_channel
impl Peer {
    pub async fn new(info: PeerInfo) -> Self {
        let state = Arc::new(Mutex::new(ConnectionState::Disconnected));
        let making_offer = Arc::new(Mutex::new(false));
        let connection = Arc::new(create_peer_connection().await.unwrap());

        // This channel communicates what peer signals (sdp offer/answer, ice candidate)
        // our tcp server should be broadcasting to the other clients
        let (sender, receiver) = mpsc::channel::<Packet>(32);

        // This will be called by WebRTC when a new data channel is created
        // It's our chance to send an offer to the peer
        connection.on_negotiation_needed(Box::new(move || {
            let connection_clone = connection.clone();
            let making_offer_clone = making_offer.clone();
            let sender_clone = sender.clone();

            Box::pin(async move {
                let flag = making_offer_clone.unwrap();
                *flag = true;

                let offer = connection_clone.create_offer(None).await.unwrap();
                connection_clone.set_local_description(offer).await.unwrap();

                let s = serde_json::to_string(&offer).unwrap();
                sender_clone.send(Packet{
                    data: Data::Offer(s),
                    from: info.our_id,
                    to: info.id.clone(),
                }).await.unwrap();
                *flag = false;
            })
        }));

        // This will be called by WebRTC when a new ICE candidate is found
        // It's our chance to send the ice candidate ot the peer
        connection.on_ice_candidate(Box::new(move |candidate: Option<RTCIceCandidate>| {
            Box::pin(async move {
                let sender_clone = sender.clone();

                if let Some(c) = candidate {
                    sender_clone.send(Packet{
                        data: Data::ICE(serde_json::to_string(&c).unwrap()),
                        from: info.our_id,
                        to: info.id.clone(),
                    }).await.unwrap();
                }
            })
        }));

        // Track the connection state
        connection.on_peer_connection_state_change(Box::new(|conn_state: RTCPeerConnectionState| {
            let mut s = state.await.unwrap();
            *s = match conn_state {
                RTCPeerConnectionState::Connecting => ConnectionState::Connecting,
                RTCPeerConnectionState::Connected => ConnectionState::Connected,
                _ => ConnectionState::Disconnected,
            };

            Box::pin(async {})
        }));

        Self {
            info,
            state,
            sender,
            receiver,
            making_offer,
            connection,
        }
    }

    async fn handle_peer_signal(&self, packet: Packet) {
        match packet.data {
            Data::Offer(offer) => {
                // Check if we're receiving an offer from a
                // peer in the middle of making one for them
                let signaling_state = self.connection.signaling_state().unwrap();
                let negotiating = signaling_state != RTCSignalingState::Stable;
                let making_offer = self.making_offer.lock().unwrap();
                let offer_collision = negotiating || *making_offer;

                // Perfect negotiation
                if offer_collision && !self.info.polite {
                    return; // Ignore the peer's offer and move forwards with our own
                }

                // Make sure we don't create an offer before setting the remote description
                if offer_collision && self.info.polite {
                    let rollback = RTCSessionDescription {
                        sdp_type: RTCSdpType::Rollback,
                        sdp: "".into(),
                    };
                    self.connection.set_local_description(rollback).await.unwrap();
                }

                let offer: RTCSessionDescription = serde_json::from_str(&offer).unwrap();
                self.connection.set_remote_description(offer).await.unwrap();

                let answer = self.connection.create_answer(None).await.unwrap();
                self.connection.set_local_description(answer).await.unwrap();

                let s = serde_json::to_string(&answer).unwrap();
                self.sender.send(Packet{
                    data: Data::Answer(s),
                    from: self.info.our_id,
                    to: packet.from,
                }).await.unwrap();
            },
            Data::Answer(answer) => {
                let answer: RTCSessionDescription = serde_json::from_str(&answer).unwrap();
                self.connection.set_remote_description(answer).await.unwrap();
            },
            Data::ICE(ice) => {
                let candidate = serde_json::from_str(&ice).unwrap();
                self.connection.add_ice_candidate(candidate).await.unwrap();
            },
            _ => panic!("unknown packet type"),
        }
    }

    // TODO: how to stop both threads?
    async fn run_client_and_server(&self) {
        // Bind to any ip on the specific port
        let socket = UdpSocket::bind("0.0.0.0:12345").await.unwrap();
        socket.set_broadcast(true).unwrap();

        // Run the server, which receives and handles peer signals
        tokio::spawn(async move {
            loop {
                let mut buffer = vec![0u8; 65536];
                let (len, addr) = socket.recv_from(&mut buffer).await.unwrap();
                buffer.truncate(len);

                let packet: Packet = serde_json::from_slice(&buffer).unwrap();
                if packet.to == self.info.our_id {
                    self.handle_peer_signal(packet).await;
                }
            }
        });

        tokio::spawn(async move {
            // Run the client, which forwards peer signals to other peers
            let broadcast_addr: SocketAddr = "255.255.255.255:1234".parse().unwrap();

            while let Some(packet) = self.receiver.recv().await {
                let msg = serde_json::to_string(&packet).unwrap();
                socket.send_to(msg, broadcast_addr).await.unwrap();
            }
        });
    }
}
