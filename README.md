# ha-whatsapp-bridge

[![HACS Custom](https://img.shields.io/badge/HACS-Custom-41BDF5.svg?style=for-the-badge)](https://github.com/hacs/integration)
[![GitHub Release](https://img.shields.io/github/v/release/Xr1H0/ha-whatsapp-bridge?style=for-the-badge&color=green)](https://github.com/Xr1H0/ha-whatsapp-bridge/releases)
[![GitHub Stars](https://img.shields.io/github/stars/Xr1H0/ha-whatsapp-bridge?style=for-the-badge&color=yellow)](https://github.com/Xr1H0/ha-whatsapp-bridge/stargazers)
[![License](https://img.shields.io/badge/License-MIT-lightgrey?style=for-the-badge)](LICENSE)

---

**Have you ever wanted to send a WhatsApp message when something happens in your smart home?** Motion detected while you're away, a window left open in the rain, or the washing machine done — delivered straight to your personal WhatsApp account, from your own Home Assistant, running 24/7 without needing your phone or Mac to be home.

This project is a **Home Assistant Add-on** that runs a WhatsApp Web bridge ([whatsmeow](https://github.com/tulir/whatsmeow)) directly on your HA PC. Pair it once with a QR code — after that, your HA can send WhatsApp messages completely autonomously.

---

## How it works

```
HA Automation          HACS Integration         Add-on (Docker)
──────────────         ────────────────         ───────────────────
notify.whatsapp_alice →  whatsapp_notify   →    Go + whatsmeow bridge
                         localhost:8080          WhatsApp Multi-Device
                        /api/send               Session in /data/
```

The Add-on implements the **WhatsApp Multi-Device protocol** — after the initial QR scan, the bridge communicates independently with WhatsApp servers. Your phone does not need to be online.

---

## Installation

### Step 1 — Install the Add-on

[![Open your Home Assistant instance and add this add-on repository.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FXr1H0%2Fha-whatsapp-bridge)

<details>
<summary>Manual steps</summary>

1. Go to **Settings → Add-ons → Add-on Store**
2. Click **⋮ (three dots)** in the top right → **Repositories**
3. Add: `https://github.com/Xr1H0/ha-whatsapp-bridge`
4. Search for **WhatsApp Bridge** and install it

</details>

### Step 2 — Start and pair WhatsApp

1. Start the **WhatsApp Bridge** add-on
2. Open `http://<your-ha-ip>:8080/qr` in a browser
3. On your phone: **WhatsApp → Linked Devices → Link a Device** → scan the QR code
4. Done — the session is saved to `/data/whatsapp.db` and survives restarts

### Step 3 — Install the HACS integration

[![Open your Home Assistant instance and open a repository inside the Home Assistant Community Store.](https://my.home-assistant.io/badges/hacs_repository.svg)](https://my.home-assistant.io/redirect/hacs_repository/?owner=Xr1H0&repository=ha-whatsapp-bridge&category=integration)

<details>
<summary>Manual steps</summary>

In HACS: **Integrations → Custom repositories** → add `https://github.com/Xr1H0/ha-whatsapp-bridge` → category: **Integration**

Or copy manually: `custom_components/whatsapp_notify/` into your HA `config/custom_components/` folder.

</details>

### Step 4 — Configure `configuration.yaml`

```yaml
notify:
  - platform: whatsapp_notify
    name: whatsapp_alice
    host: localhost      # same machine as HA
    port: 8080
    recipient: "4912345678901"   # phone number without +

  - platform: whatsapp_notify
    name: whatsapp_bob
    host: localhost
    port: 8080
    recipient: "4912345678902"
```

Restart Home Assistant after adding the config.

### Step 5 — Use in automations

```yaml
action:
  - action: notify.whatsapp_alice
    data:
      message: "Alarm ausgelöst! Bewegung im Keller."
```

---

## Requirements

- **Home Assistant OS** or **Home Assistant Supervised** (Add-on framework required)
- **HACS** for the integration (or manual install)
- A WhatsApp account for the initial QR pairing

---

## Features

- **Runs on HA directly** — no Mac, no separate server, no cloud service
- **WhatsApp Multi-Device** — phone does not need to be online after pairing
- **Persistent session** — QR scan only needed once, session stored in `/data/`
- **Browser QR page** — pair from any browser on your local network
- **Send to any number** — configure multiple recipients or send ad-hoc
- **Media support** — send images from `/config/www/` (e.g. camera snapshots)
- **Multi-arch** — runs on x86 PC, Raspberry Pi (aarch64), and armv7

---

## API

The bridge exposes a simple REST API on port 8080:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /qr` | GET | QR code pairing page (browser) |
| `GET /api/status` | GET | JSON: `{connected, paired, qr_pending}` |
| `POST /api/send` | POST | Send a message |

`POST /api/send` body:
```json
{
  "recipient": "4912345678901",
  "message": "Hello from HA!",
  "media_path": "/config/www/snapshot.jpg"
}
```

---

## Troubleshooting

**QR not loading** — Check that the add-on is running and port 8080 is accessible from your browser.

**Not connected after restart** — Check add-on logs. If the session expired, re-pair: stop the add-on, delete `/data/whatsapp.db`, restart.

**`notify.whatsapp_alice` not found** — Check that the custom component is installed and HA was restarted after adding the `configuration.yaml` entry.

**Messages not sending** — Verify `GET http://<ha-ip>:8080/api/status` returns `"connected": true`.

---

## License

MIT — see [LICENSE](LICENSE)
