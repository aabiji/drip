pub mod mdns;
pub mod p2p;

use crate::p2p::{P2PService, SafeP2PService};

#[tokio::main(flavor = "multi_thread", worker_threads = 4)]
pub async fn start_background_tasks(service: SafeP2PService) {
    P2PService::run_mdns(service.clone()).await;
}
