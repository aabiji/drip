# drip
An app that let's you share files between multiple devices.
It's like Airdrop or KDE Connect

Currently in progress...

Structure:
./ -> App using GioUI
./p2p -> Peer to peer file transfer library. Uses mDNS to find peers and WebRTC to send data.

TODO:
- general documentation about the codebase
- center the main content layout
- the mDNS search doesn't seem to be working?
- Open the file picker from java and return file contents and names

Build:
```sh
# follow the instructions here: https://gioui.org/doc/install/android
# connect to device, physical or virtual
./build.sh

# get log messages
adb logcat -s drip-debug
```
