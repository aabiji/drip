# drip
An app that let's you share files between multiple devices.
It's like Airdrop or KDE Connect

There are 2 crates:
- drip-net implements the p2p file sharing
- drip-app implements the desktop, web and mobile frontend using Dioxus

Building:
```bash
dx serve --platform desktop --package=drip-app
```
