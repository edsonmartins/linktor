"""
HTTP client utility
"""

import asyncio
from typing import Any, AsyncIterator, Callable, Optional, TypeVar

import httpx

from linktor.utils.errors import (
    LinktorError,
    NetworkError,
    TimeoutError,
    create_error_from_response,
    is_retryable_error,
)

T = TypeVar("T")


class HttpClient:
    """Synchronous HTTP client"""

    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        access_token: Optional[str] = None,
        timeout: float = 30.0,
        max_retries: int = 3,
        retry_delay: float = 1.0,
        headers: Optional[dict[str, str]] = None,
        on_token_refresh: Optional[Callable[[], str]] = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.access_token = access_token
        self.timeout = timeout
        self.max_retries = max_retries
        self.retry_delay = retry_delay
        self.custom_headers = headers or {}
        self.on_token_refresh = on_token_refresh

        self._client = httpx.Client(
            base_url=self.base_url,
            timeout=timeout,
            headers=self._build_headers(),
        )

    def _build_headers(self) -> dict[str, str]:
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
            **self.custom_headers,
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        elif self.access_token:
            headers["Authorization"] = f"Bearer {self.access_token}"
        return headers

    def _handle_response(self, response: httpx.Response) -> Any:
        request_id = response.headers.get("x-request-id")

        if response.status_code >= 400:
            try:
                body = response.json()
            except Exception:
                body = {"message": response.text}
            raise create_error_from_response(response.status_code, body, request_id)

        if response.status_code == 204:
            return None

        return response.json()

    def _request_with_retry(
        self,
        method: str,
        path: str,
        **kwargs: Any,
    ) -> Any:
        last_error: Optional[Exception] = None

        for attempt in range(self.max_retries + 1):
            try:
                response = self._client.request(method, path, **kwargs)
                return self._handle_response(response)
            except httpx.TimeoutException:
                last_error = TimeoutError("Request timeout")
            except httpx.NetworkError as e:
                last_error = NetworkError(f"Network error: {e}")
            except LinktorError as e:
                last_error = e
                if not is_retryable_error(e) or attempt == self.max_retries:
                    raise

            if attempt < self.max_retries:
                import time

                delay = self.retry_delay * (2**attempt)
                time.sleep(delay)

        if last_error:
            raise last_error

    def get(self, path: str, params: Optional[dict[str, Any]] = None) -> Any:
        return self._request_with_retry("GET", path, params=params)

    def post(
        self, path: str, data: Optional[dict[str, Any]] = None, **kwargs: Any
    ) -> Any:
        return self._request_with_retry("POST", path, json=data, **kwargs)

    def put(
        self, path: str, data: Optional[dict[str, Any]] = None, **kwargs: Any
    ) -> Any:
        return self._request_with_retry("PUT", path, json=data, **kwargs)

    def patch(
        self, path: str, data: Optional[dict[str, Any]] = None, **kwargs: Any
    ) -> Any:
        return self._request_with_retry("PATCH", path, json=data, **kwargs)

    def delete(self, path: str, **kwargs: Any) -> Any:
        return self._request_with_retry("DELETE", path, **kwargs)

    def upload(
        self,
        path: str,
        file: bytes,
        filename: str,
        field_name: str = "file",
        additional_data: Optional[dict[str, str]] = None,
    ) -> Any:
        files = {field_name: (filename, file)}
        data = additional_data or {}
        return self._request_with_retry("POST", path, files=files, data=data)

    def set_access_token(self, token: str) -> None:
        self.access_token = token
        self._client.headers["Authorization"] = f"Bearer {token}"

    def set_api_key(self, api_key: str) -> None:
        self.api_key = api_key
        self._client.headers["X-API-Key"] = api_key

    def close(self) -> None:
        self._client.close()


class AsyncHttpClient:
    """Asynchronous HTTP client"""

    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        access_token: Optional[str] = None,
        timeout: float = 30.0,
        max_retries: int = 3,
        retry_delay: float = 1.0,
        headers: Optional[dict[str, str]] = None,
        on_token_refresh: Optional[Callable[[], str]] = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.access_token = access_token
        self.timeout = timeout
        self.max_retries = max_retries
        self.retry_delay = retry_delay
        self.custom_headers = headers or {}
        self.on_token_refresh = on_token_refresh

        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            timeout=timeout,
            headers=self._build_headers(),
        )

    def _build_headers(self) -> dict[str, str]:
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
            **self.custom_headers,
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        elif self.access_token:
            headers["Authorization"] = f"Bearer {self.access_token}"
        return headers

    def _handle_response(self, response: httpx.Response) -> Any:
        request_id = response.headers.get("x-request-id")

        if response.status_code >= 400:
            try:
                body = response.json()
            except Exception:
                body = {"message": response.text}
            raise create_error_from_response(response.status_code, body, request_id)

        if response.status_code == 204:
            return None

        return response.json()

    async def _request_with_retry(
        self,
        method: str,
        path: str,
        **kwargs: Any,
    ) -> Any:
        last_error: Optional[Exception] = None

        for attempt in range(self.max_retries + 1):
            try:
                response = await self._client.request(method, path, **kwargs)
                return self._handle_response(response)
            except httpx.TimeoutException:
                last_error = TimeoutError("Request timeout")
            except httpx.NetworkError as e:
                last_error = NetworkError(f"Network error: {e}")
            except LinktorError as e:
                last_error = e
                if not is_retryable_error(e) or attempt == self.max_retries:
                    raise

            if attempt < self.max_retries:
                delay = self.retry_delay * (2**attempt)
                await asyncio.sleep(delay)

        if last_error:
            raise last_error

    async def get(self, path: str, params: Optional[dict[str, Any]] = None) -> Any:
        return await self._request_with_retry("GET", path, params=params)

    async def post(
        self, path: str, data: Optional[dict[str, Any]] = None, **kwargs: Any
    ) -> Any:
        return await self._request_with_retry("POST", path, json=data, **kwargs)

    async def put(
        self, path: str, data: Optional[dict[str, Any]] = None, **kwargs: Any
    ) -> Any:
        return await self._request_with_retry("PUT", path, json=data, **kwargs)

    async def patch(
        self, path: str, data: Optional[dict[str, Any]] = None, **kwargs: Any
    ) -> Any:
        return await self._request_with_retry("PATCH", path, json=data, **kwargs)

    async def delete(self, path: str, **kwargs: Any) -> Any:
        return await self._request_with_retry("DELETE", path, **kwargs)

    async def upload(
        self,
        path: str,
        file: bytes,
        filename: str,
        field_name: str = "file",
        additional_data: Optional[dict[str, str]] = None,
    ) -> Any:
        files = {field_name: (filename, file)}
        data = additional_data or {}
        return await self._request_with_retry("POST", path, files=files, data=data)

    async def stream(
        self, path: str, data: Optional[dict[str, Any]] = None
    ) -> AsyncIterator[dict[str, Any]]:
        async with self._client.stream("POST", path, json=data) as response:
            async for line in response.aiter_lines():
                if line.startswith("data: "):
                    json_str = line[6:].strip()
                    if json_str and json_str != "[DONE]":
                        import json

                        try:
                            yield json.loads(json_str)
                        except json.JSONDecodeError:
                            pass

    def set_access_token(self, token: str) -> None:
        self.access_token = token
        self._client.headers["Authorization"] = f"Bearer {token}"

    def set_api_key(self, api_key: str) -> None:
        self.api_key = api_key
        self._client.headers["X-API-Key"] = api_key

    async def close(self) -> None:
        await self._client.aclose()


__all__ = ["HttpClient", "AsyncHttpClient"]
