use crate::p2p::PeerToPeerService;

pub mod mdns;
pub mod p2p;

use std::sync::Arc;
use tokio::sync::Mutex;

#[tokio::main(flavor = "multi_thread", worker_threads = 4)]
pub async fn start_background_tasks(service: Arc<Mutex<PeerToPeerService>>) {
    p2p::PeerToPeerService::run_mdns(service.clone()).await;
}
