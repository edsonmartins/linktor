"""Tests for the Channels resource (wire alignment: QR, pair, credentials)."""

from unittest.mock import Mock

from linktor.client import ChannelsResource


def _http(responses):
    """Build a mock HttpClient whose verbs return preset envelopes and record calls."""
    http = Mock()
    calls = []

    def make(verb):
        def fn(path, data=None, **kwargs):
            calls.append({"verb": verb, "path": path, "data": data})
            return responses.get(f"{verb} {path}")
        return fn

    http.get = Mock(side_effect=lambda path, params=None: (
        calls.append({"verb": "GET", "path": path, "data": None}),
        responses.get(f"GET {path}"),
    )[1])
    http.post = Mock(side_effect=make("POST"))
    http.put = Mock(side_effect=make("PUT"))
    http.delete = Mock(side_effect=make("DELETE"))
    return http, calls


def test_connect_unwraps_envelope_and_surfaces_qr():
    http, _ = _http({
        "POST /channels/ch1/connect": {
            "success": True,
            "data": {
                "channel": {
                    "id": "ch1", "tenant_id": "t", "name": "wa",
                    "type": "whatsapp", "connection_status": "connecting",
                },
                "qr_code": "QR-PAYLOAD-123",
                "expires_in": 60,
            },
        }
    })
    res = ChannelsResource(http).connect("ch1")
    assert res.qr_code == "QR-PAYLOAD-123"
    assert res.expires_in == 60
    assert res.channel.id == "ch1"
    assert res.channel.connection_status.value == "connecting"


def test_create_sends_credentials():
    http, calls = _http({
        "POST /channels": {
            "success": True,
            "data": {
                "id": "ch9", "tenant_id": "t", "name": "wa",
                "type": "whatsapp", "connection_status": "disconnected",
            },
        }
    })
    ch = ChannelsResource(http).create(
        name="wa", type="whatsapp", credentials={"access_token": "secret"}
    )
    assert ch.id == "ch9"
    assert calls[0]["data"]["credentials"]["access_token"] == "secret"


def test_update_uses_put():
    http, calls = _http({
        "PUT /channels/ch1": {
            "success": True,
            "data": {
                "id": "ch1", "tenant_id": "t", "name": "renamed",
                "type": "whatsapp", "connection_status": "disconnected",
            },
        }
    })
    ChannelsResource(http).update("ch1", name="renamed")
    assert calls[0]["verb"] == "PUT"
    assert calls[0]["path"] == "/channels/ch1"


def test_request_pair_code():
    http, calls = _http({
        "POST /channels/ch1/pair": {"success": True, "data": {"pair_code": "ABCD-1234"}}
    })
    res = ChannelsResource(http).request_pair_code("ch1", "+5511999999999")
    assert res.pair_code == "ABCD-1234"
    assert calls[0]["data"]["phone_number"] == "+5511999999999"


def test_list_unwraps_plain_array():
    http, _ = _http({
        "GET /channels": {
            "success": True,
            "data": [
                {"id": "a", "tenant_id": "t", "name": "x", "type": "whatsapp",
                 "connection_status": "connected"},
                {"id": "b", "tenant_id": "t", "name": "y", "type": "telegram",
                 "connection_status": "disconnected"},
            ],
        }
    })
    channels = ChannelsResource(http).list()
    assert [c.id for c in channels] == ["a", "b"]
