# Changelog

## 1.0.3

- Change default port from 8080 to 8456 to avoid conflicts with common services

## 1.0.2

- Add CHANGELOG.md

## 1.0.1

- Fix QR pairing page: QR code is now generated server-side as PNG — no external CDN required
- Resolves blank/grey box shown instead of QR code in HA containers

## 1.0.0

- Initial release
- WhatsApp Web bridge using whatsmeow (Multi-Device protocol)
- REST API on port 8080: `/qr`, `/api/send`, `/api/status`
- Persistent session stored in `/data/whatsapp.db`
- Multi-arch support: amd64, aarch64, armv7
- HACS integration via `notify.whatsapp_notify` platform
