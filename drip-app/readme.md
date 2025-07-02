

Backend threads:
- Use a ping/pong model to discover and signal to peers
  Ping/Pong
- Once a connection between peers has been established,
  use WebRTC to transfer files

Frontend:
- Settings
    - Downloads folder
    - Theme
    - Credits
- Clients
    - Info on connected peers
    - Peers near you --> connect to one? (should we make a settings option to auto connect???)
- Files
    - Show transfered files
      Transfered files, files pending transfering...
      Show info on who did the transfering, the file name, file size
      Show file preview if capable (images, video thumbnails).
      else just regular placeholders for different file types

- Show transfered files (dow)

### Tailwind
1. Install npm: https://docs.npmjs.com/downloading-and-installing-node-js-and-npm
2. Install the Tailwind CSS CLI: https://tailwindcss.com/docs/installation
3. Run the following command in the root of the project to start the Tailwind CSS compiler:

```bash
npx tailwindcss -i ./tailwind.css -o ./assets/tailwind.css --watch
```

### Serving Your App

```bash
dx serve --platform desktop
```
