# Homelab deployment (LAN)

Runs shmeeload.xyz on the home network with self-hosted HTTPS. No public exposure,
no third-party services.

## Topology

```
phone / laptop  --https://shmee.lan-->  Caddy (:443, internal CA)  -->  shmeeload (:8080)
       \                                        on the Pi (192.168.68.56)
        \--http://192.168.68.56:8080----------------------------------/ (direct HTTP fallback)
```

- **shmeeload**: the Go server, in a container, unprivileged, on `:8080`.
- **Caddy**: reverse proxy terminating TLS with its own internal CA (no public ACME,
  so no domain is required). Serves `https://shmee.lan`.
- **Pi-hole**: holds the local DNS record `shmee.lan -> 192.168.68.56` so devices on
  the LAN resolve the name. A hostname (not a bare IP) is required so browsers send
  SNI and Caddy can match the cert.

## Deploy / update

From the repo root, sync to the Pi and rebuild:

```sh
rsync -az --delete --exclude .git --exclude web_res/node_modules \
  --exclude web_res/dist --exclude static --exclude server \
  ./ cozart@192.168.68.56:/opt/stacks/shmeeload/
ssh cozart@192.168.68.56 'cd /opt/stacks/shmeeload/deploy && docker compose up -d --build'
```

`restart: unless-stopped` + Docker enabled on boot means the stack comes back after a reboot.

## Trust the internal CA on a device (one time, per device)

Caddy's root cert is at `/data/caddy/pki/authorities/local/root.crt` inside the caddy
container. Export it:

```sh
ssh cozart@192.168.68.56 'docker exec caddy cat /data/caddy/pki/authorities/local/root.crt' > caddy-root-pi.crt
```

- **iOS**: AirDrop/email the `.crt` to the phone, open it, then
  Settings → General → VPN & Device Management → install the profile, then
  Settings → General → About → Certificate Trust Settings → enable full trust for
  "Caddy Local Authority". Browse `https://shmee.lan`.
- **macOS**: `open caddy-root-pi.crt`, add to login keychain, set to "Always Trust".
- **Android**: Settings → Security → Encryption & credentials → Install a certificate →
  CA certificate. Browse `https://shmee.lan`.

The root cert persists in the `caddy_data` volume, so it survives restarts and you only
install it on each device once.

## Notes

- The site renders with origin-relative URLs, so the same deployment works at
  `https://shmee.lan`, `http://192.168.68.56:8080`, or any future hostname with no
  reconfiguration.
- The memory cap in `docker-compose.yml` is currently discarded: the Pi's kernel needs
  the memory cgroup enabled (`cgroup_enable=memory cgroup_memory=1` on the kernel
  cmdline, then reboot). The CPU cap works without it.
