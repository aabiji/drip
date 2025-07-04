use std::net::IpAddr;

#[derive(Clone)]
pub enum ConnectionState {
    Disconnected,
    Connecting,
    Connected,
}

#[derive(Clone)]
pub struct Peer {
    pub ip: IpAddr,
    pub id: String,
    pub is_mobile: bool,
    pub state: ConnectionState,
}

impl Peer {
    pub fn new(ip: IpAddr, id: String, is_mobile: bool) -> Self {
        Self {
            ip,
            id,
            is_mobile,
            state: ConnectionState::Disconnected,
        }
    }
}
