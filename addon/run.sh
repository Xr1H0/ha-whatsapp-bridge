#!/usr/bin/with-contenv bashio

PORT=$(bashio::config 'port' '8080')
LOG_LEVEL=$(bashio::config 'log_level' 'info')

export PORT="${PORT}"
export DATA_PATH="/data"
export LOG_LEVEL="${LOG_LEVEL}"

bashio::log.info "Starting WhatsApp Bridge on port ${PORT}"
bashio::log.info "Data directory: ${DATA_PATH}"
bashio::log.info "Navigate to http://<ha-ip>:${PORT}/qr to pair WhatsApp"

exec /app/whatsapp-bridge
