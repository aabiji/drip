use std::net::{IpAddr, SocketAddr};
use std::sync::Arc;
use std::boxed::Box;

use serde::{Deserialize, Serialize};

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

#[derive(Clone, Deserialize, Serialize)]
enum PacketType {
    Offer(String),
    Answer(String),
    ICE(String),
}

#[derive(Clone, Deserialize, Serialize)]
struct Packet {
    data: PacketType,
    from: String,
    to: String,
}

#[derive(Clone)]
pub enum ConnectionState {
    Disconnected,
    Connecting,
    Connected,
}

#[derive(Clone)]
pub struct PeerInfo {
    pub id: String,
    pub our_id: String,
    pub ip: IpAddr,
    pub mobile: bool,
    pub polite: bool,
    pub state: Arc<Mutex<ConnectionState>>,
}

pub struct Peer {
    pub info: PeerInfo,

    sender: Sender<Packet>,
    receiver: Arc<Mutex<Receiver<Packet>>>,

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

impl Peer {
    pub async fn new(info: PeerInfo) -> Self {
        let making_offer = Arc::new(Mutex::new(false));
        let connection = Arc::new(create_peer_connection().await.unwrap());

        let (sender, receiver) = mpsc::channel::<Packet>(32);
        let receiver = Arc::new(Mutex::new(receiver));

        let connection_clone = connection.clone();
        let making_offer_clone = making_offer.clone();
        let sender_clone = sender.clone();
        let info_clone = info.clone();

        connection.on_negotiation_needed(Box::new(move || {
            let connection = connection_clone.clone();
            let making_offer = making_offer_clone.clone();
            let sender = sender_clone.clone();
            let info = info_clone.clone();

            Box::pin(async move {
                let mut flag = making_offer.lock().await;
                *flag = true;

                let offer = connection.create_offer(None).await.unwrap();
                connection.set_local_description(offer.clone()).await.unwrap();

                let s = serde_json::to_string(&offer).unwrap();
                sender.send(Packet {
                    data: PacketType::Offer(s),
                    from: info.our_id.clone(),
                    to: info.id.clone(),
                }).await.unwrap();

                *flag = false;
            })
        }));

        let connection_clone = connection.clone();
        let sender_clone = sender.clone();
        let info_clone = info.clone();

        connection.on_ice_candidate(Box::new(move |candidate: Option<RTCIceCandidate>| {
            let sender = sender_clone.clone();
            let info = info_clone.clone();
            Box::pin(async move {
                if let Some(c) = candidate {
                    sender.send(Packet {
                        data: PacketType::ICE(serde_json::to_string(&c).unwrap()),
                        from: info.our_id.clone(),
                        to: info.id.clone(),
                    }).await.unwrap();
                }
            })
        }));

        let state_clone = info.state.clone();
        connection.on_peer_connection_state_change(Box::new(move |conn_state: RTCPeerConnectionState| {
            let state = state_clone.clone();
            Box::pin(async move {
                let mut lock = state.lock().await;
                *lock = match conn_state {
                    RTCPeerConnectionState::Connecting => ConnectionState::Connecting,
                    RTCPeerConnectionState::Connected => ConnectionState::Connected,
                    _ => ConnectionState::Disconnected,
                };
            })
        }));

        Self {
            info,
            sender,
            receiver,
            making_offer,
            connection,
        }
    }

    async fn handle_peer_signal(self: &Arc<Self>, packet: Packet) {
        match packet.data {
            PacketType::Offer(offer) => {
                let signaling_state = self.connection.signaling_state();
                let negotiating = signaling_state != RTCSignalingState::Stable;
                let making_offer = self.making_offer.lock().await;
                let offer_collision = negotiating || *making_offer;

                if offer_collision && !self.info.polite {
                    return;
                }

                if offer_collision && self.info.polite {
                    let mut rollback = RTCSessionDescription::default();
                    rollback.sdp_type = RTCSdpType::Rollback;
                    rollback.sdp = "".into();
                    self.connection.set_local_description(rollback).await.unwrap();
                }

                let offer: RTCSessionDescription = serde_json::from_str(&offer).unwrap();
                self.connection.set_remote_description(offer).await.unwrap();

                let answer = self.connection.create_answer(None).await.unwrap();
                self.connection.set_local_description(answer.clone()).await.unwrap();

                let s = serde_json::to_string(&answer).unwrap();
                self.sender.send(Packet {
                    data: PacketType::Answer(s),
                    from: self.info.our_id.clone(),
                    to: packet.from,
                }).await.unwrap();
            },
            PacketType::Answer(answer) => {
                let answer: RTCSessionDescription = serde_json::from_str(&answer).unwrap();
                self.connection.set_remote_description(answer).await.unwrap();
            },
            PacketType::ICE(ice) => {
                let candidate = serde_json::from_str(&ice).unwrap();
                self.connection.add_ice_candidate(candidate).await.unwrap();
            },
        }
    }

    pub async fn run_client_and_server(self: Arc<Self>) {
        let socket = Arc::new(UdpSocket::bind("0.0.0.0:12345").await.unwrap());
        socket.set_broadcast(true).unwrap();

        let self_recv = self.clone();
        let socket_recv = socket.clone();
        tokio::spawn(async move {
            loop {
                let mut buffer = vec![0u8; 65536];
                let (len, _) = socket_recv.recv_from(&mut buffer).await.unwrap();
                buffer.truncate(len);

                let packet: Packet = serde_json::from_slice(&buffer).unwrap();
                if packet.to == self_recv.info.our_id {
                    self_recv.handle_peer_signal(packet).await;
                }
            }
        });

        let self_send = self.clone();
        let socket_send = socket.clone();
        tokio::spawn(async move {
            let broadcast_addr: SocketAddr = "255.255.255.255:1234".parse().unwrap();
            let mut receiver = self_send.receiver.lock().await;
            while let Some(packet) = receiver.recv().await {
                let msg = serde_json::to_string(&packet).unwrap();
                socket_send.send_to(msg.as_bytes(), broadcast_addr).await.unwrap();
            }
        });
    }
}
