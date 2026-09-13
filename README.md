# ha-whatsapp-bridge

WhatsApp Web bridge for Home Assistant — send WhatsApp messages from HA automations and scripts, running autonomously on the HA x86 PC without needing a Mac or phone.

## Architecture

```
HA Add-on (Docker)          HACS Integration
────────────────            ────────────────
Go + whatsmeow              notify.whatsapp_notify
WhatsApp Multi-Device  ←→   configuration.yaml entries
REST API :8080              calls localhost:8080/api/send
Session in /data/
```

The Add-on runs a Go-based WhatsApp Web client (using [whatsmeow](https://github.com/tulir/whatsmeow)) directly on the HA PC. After a one-time QR code scan with your phone, the bridge maintains an independent session — your phone does not need to be home.

## Setup

### 1. Install the Add-on

In Home Assistant: **Settings → Add-ons → Add-on Store → ⋮ → Repositories**

Add this URL:
```
https://github.com/Xr1H0/ha-whatsapp-bridge
```

Then install **WhatsApp Bridge** from the list.

### 2. Pair WhatsApp

1. Start the add-on
2. Open `http://<your-ha-ip>:8080/qr` in a browser
3. Open WhatsApp on your phone → **Linked Devices** → **Link a Device** → scan the QR code
4. Done — session is saved to `/data/whatsapp.db` and survives restarts

### 3. Install the HACS Integration

In HACS: **Integrations → Custom repositories** → add this repo URL → category: Integration.

Or install manually: copy `custom_components/whatsapp_notify/` into your HA `config/custom_components/` folder.

### 4. Configure `configuration.yaml`

```yaml
notify:
  - platform: whatsapp_notify
    name: whatsapp_alice
    host: localhost      # or 127.0.0.1 — same machine as HA
    port: 8080
    recipient: "4912345678901"   # phone number without +

  - platform: whatsapp_notify
    name: whatsapp_bob
    host: localhost
    port: 8080
    recipient: "4912345678902"
```

Restart HA after adding the config.

### 5. Use in automations

```yaml
action:
  - action: notify.whatsapp_alice
    data:
      message: "Alarm ausgelöst!"
```

Send to a different number ad-hoc:
```yaml
action:
  - action: notify.whatsapp_alice
    target:
      entity_id: "491234567890"
    data:
      message: "Hallo!"
```

## API

The bridge exposes a simple REST API:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /qr` | GET | QR code pairing page (browser) |
| `GET /api/status` | GET | JSON status: connected, paired, qr_pending |
| `POST /api/send` | POST | Send a message |

`POST /api/send` body:
```json
{
  "recipient": "4912345678901",
  "message": "Hello from HA!",
  "media_path": "/config/www/snapshot.jpg"
}
```

## Troubleshooting

- **QR not loading**: Check add-on is running, port 8080 is accessible
- **Not connected after restart**: Check add-on logs, session may need re-pairing
- **Messages not sending**: Check `notify.whatsapp_alice` exists in HA (Developer Tools → States)
- **Re-pair**: Stop add-on, delete `/data/whatsapp.db`, restart
