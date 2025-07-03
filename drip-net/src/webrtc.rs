use std::sync::Arc;
use webrtc::api::interceptor_registry::register_default_interceptors;
use webrtc::api::media_engine::MediaEngine;
use webrtc::api::APIBuilder;
use webrtc::data_channel::data_channel_message::DataChannelMessage;
use webrtc::error::Result;
use webrtc::interceptor::registry::Registry;
use webrtc::peer_connection::configuration::RTCConfiguration;
use webrtc::peer_connection::peer_connection_state::RTCPeerConnectionState;
use webrtc::peer_connection::sdp::session_description::RTCSessionDescription;
use webrtc::peer_connection::RTCPeerConnection;

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

async fn do_peer_signalling(peer_connection: &RTCPeerConnection) -> Result<()> {
    // Create an offer
    let offer = peer_connection.create_offer(None).await?;
    peer_connection.set_local_description(offer).await?;

    // Set the answer
    let str = "TODO: get this from our tcp connection with peer!";
    let answer: RTCSessionDescription = serde_json::from_str(str)
        .map_err(|e| webrtc::Error::new(format!("serde_json error: {}", e)))?;
    peer_connection.set_remote_description(answer).await?;

    // Create channel that is blocked until ICE Gathering is complete
    let mut gather_complete = peer_connection.gathering_complete_promise().await;

    // Block until ICE Gathering is complete, disabling trickle ICE
    // we do this because we only want to exchange one signaling message
    let _ = gather_complete.recv().await;

    Ok(())
}

async fn handle_data_channel() -> Result<()> {
    let peer_connection = crate::webrtc::create_peer_connection().await.unwrap();
    crate::webrtc::do_peer_signalling(&peer_connection).await?;

    let data_channel = peer_connection.create_data_channel("data", None).await?;

    // Send data
    let d1 = Arc::clone(&data_channel);
    data_channel.on_open(Box::new(move || {
        let d2 = Arc::clone(&d1);
        Box::pin(async move {
            let bytes = bytes::Bytes::from("hello!");
            d2.send(&bytes).await.unwrap();
        })
    }));

    // Receive data
    data_channel.on_message(Box::new(move |msg: DataChannelMessage| {
        let message = msg.data.to_vec();
        println!("message: {:?}", message);
        Box::pin(async {})
    }));

    // Handle peer disconnects
    peer_connection.on_peer_connection_state_change(Box::new(move |s: RTCPeerConnectionState| {
        let channel = data_channel.clone();
        if s == RTCPeerConnectionState::Failed {
            Box::pin(async move {
                channel.close().await.unwrap();
                println!("Peer Connection has gone to failed exiting");
            })
        } else {
            Box::pin(async {})
        }
    }));

    peer_connection.close().await?;

    Ok(())
}

#[cfg(test)]
mod tests {
    #[tokio::test]
    async fn run_example() {
        crate::webrtc::handle_data_channel().await.unwrap();
    }
}
