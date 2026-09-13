import logging
import requests
import voluptuous as vol

from homeassistant.components.notify import (
    ATTR_TARGET,
    PLATFORM_SCHEMA,
    BaseNotificationService,
)
from homeassistant.const import CONF_HOST, CONF_PORT
import homeassistant.helpers.config_validation as cv

_LOGGER = logging.getLogger(__name__)

CONF_RECIPIENT = "recipient"
DEFAULT_PORT = 8080

PLATFORM_SCHEMA = PLATFORM_SCHEMA.extend(
    {
        vol.Required(CONF_HOST): cv.string,
        vol.Optional(CONF_PORT, default=DEFAULT_PORT): cv.port,
        vol.Optional(CONF_RECIPIENT): cv.string,
    }
)


def get_service(hass, config, discovery_info=None):
    return WhatsAppNotificationService(
        host=config[CONF_HOST],
        port=config[CONF_PORT],
        default_recipient=config.get(CONF_RECIPIENT),
    )


class WhatsAppNotificationService(BaseNotificationService):
    def __init__(self, host, port, default_recipient):
        self._url = f"http://{host}:{port}/api/send"
        self._default_recipient = default_recipient

    def send_message(self, message="", **kwargs):
        targets = kwargs.get(ATTR_TARGET)
        if not targets and self._default_recipient:
            targets = [self._default_recipient]
        if not targets:
            _LOGGER.error("whatsapp_notify: no recipient — set 'recipient' in config or pass via target")
            return

        for recipient in targets:
            try:
                resp = requests.post(
                    self._url,
                    json={"recipient": str(recipient), "message": message},
                    timeout=10,
                )
                resp.raise_for_status()
                result = resp.json()
                if not result.get("success"):
                    _LOGGER.error("whatsapp_notify: send failed for %s: %s", recipient, result.get("message"))
                else:
                    _LOGGER.debug("whatsapp_notify: sent to %s", recipient)
            except requests.exceptions.ConnectionError:
                _LOGGER.error("whatsapp_notify: bridge not reachable at %s", self._url)
            except Exception as exc:
                _LOGGER.error("whatsapp_notify: error for %s: %s", recipient, exc)
