use std::collections::HashMap;
use std::net::SocketAddr;
use std::result::Result;
use std::net::UdpSocket;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use std::sync::{Arc, Mutex};

#[derive(Debug)]
pub struct Packet {
    start_bytes: [u8; 8],
    version: u8,
    timestamp: u64,
    device_name_length: u8,
    device_name: String,
}

impl Packet {
    const PACKET_START: [u8; 8] = [b'D', b'R', b'I', b'P', b'P', b'I', b'N', b'G'];
    const CURRENT_VERSION: u8 = 1;

    fn new(device_name: String, timestamp: u64) -> Packet {
        Self {
            start_bytes: Self::PACKET_START,
            version: Self::CURRENT_VERSION,
            device_name_length: device_name.len() as u8,
            device_name,
            timestamp
        }
    }

    fn deserialize(data: &[u8]) -> Result<Packet, Box<dyn std::error::Error>> {
        if !data.starts_with(&Self::PACKET_START) {
            return Err("Unrecognized packet".into());
        }

        let version = data[8];
        if version != Self::CURRENT_VERSION {
            return Err("Unrecognized packet".into());
        }

        let timestamp = u64::from_le_bytes(data[9..17].try_into()?);
        let device_name_length = data[17];
        if data.len() as u8 != 18 + device_name_length {
            return Err("Malformed packet".into());
        }
        let slice = &data[18..18 + device_name_length as usize];
        let device_name = std::str::from_utf8(slice)?.to_string();

        Ok(Packet{
            start_bytes: Self::PACKET_START,
            version,
            timestamp,
            device_name_length,
            device_name
        })
    }

    fn serialize(&self) -> Vec<u8> {
        let mut buffer = Vec::new();

        buffer.extend_from_slice(&self.start_bytes);
        buffer.push(self.version);
        buffer.extend_from_slice(&self.timestamp.to_le_bytes());
        buffer.push(self.device_name_length);
        buffer.extend_from_slice(self.device_name.as_bytes());

        buffer
    }
}

// Map each peer's address to the most recent packet they've sent
pub type Peers = Arc<Mutex<HashMap<SocketAddr, Packet>>>;

pub fn new_peers() -> Peers { Arc::new(Mutex::new(HashMap::new())) }

// Periodically ping, signialling to other devices
// that we're still connected to the network. Also
// maintain the list of connected peers
pub fn run_client(peers: Peers) -> std::io::Result<()> {
    println!("client running!");

    let socket = UdpSocket::bind("0.0.0.0:0")?;
    socket.set_broadcast(true)?;

    let peer_disconnect_timeout = 3.0 * 60.0; // 3 minutes
    let device_name = whoami::devicename();

    loop {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs())
            .map_err(std::io::Error::other)?;

        let packet = Packet::new(device_name.clone(), timestamp);
        // Broadcast the data to all the devices on the local network
        socket.send_to(&packet.serialize(), "255.255.255.255:1234")?;

        // Remove disconnected peers
        peers.lock().unwrap().retain(|_, peer| {
            let time = UNIX_EPOCH + Duration::from_secs(peer.timestamp);
            let duration = match time.elapsed() {
                Ok(d) => d,
                Err(_) => return false,
            };
            // If we haven't gotten a ping from the peer in a while,
            // we simply assume they're disconnected from the network
            duration.as_secs_f64() < peer_disconnect_timeout
        });

        // ping every minute
        //std::thread::sleep(Duration::from_secs(60));
        std::thread::sleep(Duration::from_millis(1000));
    }
}

// Receive and record the pings from other devices on the network
pub fn run_server(peers: Peers) -> std::io::Result<()> {
    println!("server running!");

    let ourselves = whoami::devicename();
    let socket = UdpSocket::bind("0.0.0.0:1234")?;
    let mut buffer = [0; 4096];

    loop {
        let (amount, addr) = socket.recv_from(&mut buffer)?;

        let packet = match Packet::deserialize(&buffer[..amount]) {
            Ok(p) => p,
            Err(_)  => continue, // simply ignore invalid packets
        };

        if packet.device_name != ourselves {
            peers.lock().unwrap().insert(addr, packet);
        }
    }
}
