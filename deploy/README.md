# Homelab deployment (LAN)

Runs shmeeload.xyz on the home network as a single self-contained Go binary that
terminates TLS itself. No reverse proxy, no third-party services.

## Topology

```
phone / laptop  --https://shmee.lan-->  shmeeload binary (:443, TLS + HTTP/2)
                                          on the Pi (192.168.68.56)
```

- The binary **embeds the whole site** (`//go:embed`) and serves it with gzip +
  ETag caching. The chat websocket and `/donate` are the only dynamic routes.
- It terminates TLS directly with a **self-signed certificate** for `shmee.lan`,
  generated on first boot into the `shmee_tls` volume (`/data`) and reused after.
- **Pi-hole** holds the local DNS record `shmee.lan -> 192.168.68.56` so devices
  on the LAN resolve the name.

## Deploy / update

From the repo root, sync to the Pi and rebuild:

```sh
rsync -az --delete --exclude .git --exclude web_res/node_modules \
  --exclude site/static/js --exclude server \
  ./ cozart@192.168.68.56:/opt/stacks/shmeeload/
ssh cozart@192.168.68.56 'cd /opt/stacks/shmeeload/deploy && docker compose up -d --build'
```

The TypeScript is compiled into `site/static/js` and embedded into the binary at
build time; the runtime image ships only the binary (+ config + cert volume).
`restart: unless-stopped` + Docker enabled on boot bring it back after a reboot.

## Trust the certificate on a device (one time, per device)

Export the self-signed cert from the volume:

```sh
ssh cozart@192.168.68.56 'docker exec shmeeload cat /data/cert.pem' > shmee-cert.pem
```

- **iOS**: AirDrop/email the `.pem`, install the profile (Settings → General →
  VPN & Device Management), then enable full trust under Settings → General →
  About → Certificate Trust Settings. Browse `https://shmee.lan`.
- **macOS**: `open shmee-cert.pem`, add to the login keychain, set "Always Trust".
- **Android**: Settings → Security → Encryption & credentials → Install a
  certificate → CA certificate.

The cert lives in the `shmee_tls` Docker volume, so it survives restarts and you
only trust it once per device. It is valid for 10 years.

## Notes

- The container runs read-only and unprivileged (`cap_drop: ALL`, only
  `NET_BIND_SERVICE` to bind :443), with the cert volume and `/tmp` writable.
- Static assets are served with precomputed gzip and content-hash ETags
  (`If-None-Match` → 304). `cacheMaxAge` in the config tunes `Cache-Control`.
- CI (build/test/lint/vuln) runs on a self-hosted x86 runner, not GitHub-hosted.
