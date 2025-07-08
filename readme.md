# drip
An app that let's you share files between multiple devices.
It's like Airdrop or KDE Connect

There are 2 packages:
- drip-net implements the p2p file sharing
- drip-app implements the desktop, web and mobile frontend using Fyne

Rational:
I'm not enjoying using Rust at all. And since side projects
are supposed to be fun, I'm going to switch to a language I
enjoy more. Use Fyne for the gui, and other packages for the
networking. Go also has channels and goroutines builtin, which
is perfect for our task. I should have used Go from the very start.

To port, work on the networking package first.
When we're finished the basic networking (peer connections, file transfers),
then work on the gui and other miscellanious code.

TODO: imporve the mdns peer discovery

icons from here:
https://www.svgrepo.com/collection/jtb-variety-thin-icons/
