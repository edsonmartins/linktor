"""Tests for LinktorClient"""

import pytest
from unittest.mock import Mock, patch, MagicMock
from linktor.client import LinktorClient, LinktorAsyncClient


class TestLinktorClient:
    """Tests for LinktorClient"""

    def test_create_client_with_defaults(self):
        """Should create client with default config"""
        client = LinktorClient()

        assert client is not None
        assert client.auth is not None
        assert client.conversations is not None
        assert client.contacts is not None
        assert client.channels is not None
        assert client.bots is not None
        assert client.ai is not None
        assert client.knowledge_bases is not None
        assert client.flows is not None
        assert client.analytics is not None

    def test_create_client_with_api_key(self):
        """Should create client with API key"""
        client = LinktorClient(api_key="test-api-key")

        assert client is not None

    def test_create_client_with_access_token(self):
        """Should create client with access token"""
        client = LinktorClient(access_token="test-access-token")

        assert client is not None

    def test_create_client_with_custom_config(self):
        """Should create client with custom configuration"""
        client = LinktorClient(
            base_url="https://custom.api.com",
            api_key="test-api-key",
            timeout=60.0,
            max_retries=5,
            retry_delay=2.0,
            headers={"X-Custom-Header": "custom-value"},
        )

        assert client is not None

    def test_set_api_key(self):
        """Should update API key"""
        client = LinktorClient()

        # Should not raise
        client.set_api_key("new-api-key")

    def test_set_access_token(self):
        """Should update access token"""
        client = LinktorClient()

        # Should not raise
        client.set_access_token("new-access-token")

    def test_close(self):
        """Should close client connections"""
        client = LinktorClient()

        # Should not raise
        client.close()

    def test_context_manager(self):
        """Should work as context manager"""
        with LinktorClient() as client:
            assert client is not None


class TestLinktorAsyncClient:
    """Tests for LinktorAsyncClient"""

    def test_create_async_client_with_defaults(self):
        """Should create async client with default config"""
        client = LinktorAsyncClient()

        assert client is not None

    def test_create_async_client_with_api_key(self):
        """Should create async client with API key"""
        client = LinktorAsyncClient(api_key="test-api-key")

        assert client is not None

    def test_create_async_client_with_custom_config(self):
        """Should create async client with custom configuration"""
        client = LinktorAsyncClient(
            base_url="https://custom.api.com",
            api_key="test-api-key",
            timeout=60.0,
            max_retries=5,
            retry_delay=2.0,
        )

        assert client is not None


class TestAuthResource:
    """Tests for AuthResource"""

    def test_auth_resource_exists(self):
        """Should have auth resource"""
        client = LinktorClient()

        assert client.auth is not None
        assert hasattr(client.auth, "login")
        assert hasattr(client.auth, "logout")
        assert hasattr(client.auth, "refresh_token")
        assert hasattr(client.auth, "get_current_user")


class TestConversationsResource:
    """Tests for ConversationsResource"""

    def test_conversations_resource_exists(self):
        """Should have conversations resource"""
        client = LinktorClient()

        assert client.conversations is not None
        assert hasattr(client.conversations, "list")
        assert hasattr(client.conversations, "get")
        assert hasattr(client.conversations, "update")
        assert hasattr(client.conversations, "send_message")
        assert hasattr(client.conversations, "send_text")
        assert hasattr(client.conversations, "list_messages")
        assert hasattr(client.conversations, "resolve")
        assert hasattr(client.conversations, "assign")


class TestContactsResource:
    """Tests for ContactsResource"""

    def test_contacts_resource_exists(self):
        """Should have contacts resource"""
        client = LinktorClient()

        assert client.contacts is not None
        assert hasattr(client.contacts, "list")
        assert hasattr(client.contacts, "get")
        assert hasattr(client.contacts, "create")
        assert hasattr(client.contacts, "update")
        assert hasattr(client.contacts, "delete")
        assert hasattr(client.contacts, "search")


class TestChannelsResource:
    """Tests for ChannelsResource"""

    def test_channels_resource_exists(self):
        """Should have channels resource"""
        client = LinktorClient()

        assert client.channels is not None
        assert hasattr(client.channels, "list")
        assert hasattr(client.channels, "get")
        assert hasattr(client.channels, "create")
        assert hasattr(client.channels, "update")
        assert hasattr(client.channels, "delete")
        assert hasattr(client.channels, "connect")
        assert hasattr(client.channels, "disconnect")


class TestBotsResource:
    """Tests for BotsResource"""

    def test_bots_resource_exists(self):
        """Should have bots resource"""
        client = LinktorClient()

        assert client.bots is not None
        assert hasattr(client.bots, "list")
        assert hasattr(client.bots, "get")
        assert hasattr(client.bots, "create")
        assert hasattr(client.bots, "update")
        assert hasattr(client.bots, "delete")
        assert hasattr(client.bots, "activate")
        assert hasattr(client.bots, "deactivate")


class TestAIResource:
    """Tests for AIResource"""

    def test_ai_resource_exists(self):
        """Should have AI resource with sub-resources"""
        client = LinktorClient()

        assert client.ai is not None
        assert client.ai.agents is not None
        assert client.ai.completions is not None
        assert client.ai.embeddings is not None

    def test_agents_subresource(self):
        """Should have agents sub-resource"""
        client = LinktorClient()

        assert hasattr(client.ai.agents, "list")
        assert hasattr(client.ai.agents, "get")
        assert hasattr(client.ai.agents, "create")
        assert hasattr(client.ai.agents, "update")
        assert hasattr(client.ai.agents, "delete")
        assert hasattr(client.ai.agents, "invoke")

    def test_completions_subresource(self):
        """Should have completions sub-resource"""
        client = LinktorClient()

        assert hasattr(client.ai.completions, "create")
        assert hasattr(client.ai.completions, "complete")

    def test_embeddings_subresource(self):
        """Should have embeddings sub-resource"""
        client = LinktorClient()

        assert hasattr(client.ai.embeddings, "create")
        assert hasattr(client.ai.embeddings, "embed")


class TestKnowledgeBasesResource:
    """Tests for KnowledgeBasesResource"""

    def test_knowledge_bases_resource_exists(self):
        """Should have knowledge bases resource"""
        client = LinktorClient()

        assert client.knowledge_bases is not None
        assert hasattr(client.knowledge_bases, "list")
        assert hasattr(client.knowledge_bases, "get")
        assert hasattr(client.knowledge_bases, "create")
        assert hasattr(client.knowledge_bases, "update")
        assert hasattr(client.knowledge_bases, "delete")
        assert hasattr(client.knowledge_bases, "query")
        assert hasattr(client.knowledge_bases, "search")
        assert hasattr(client.knowledge_bases, "list_documents")
        assert hasattr(client.knowledge_bases, "upload_document")
        assert hasattr(client.knowledge_bases, "delete_document")


class TestFlowsResource:
    """Tests for FlowsResource"""

    def test_flows_resource_exists(self):
        """Should have flows resource"""
        client = LinktorClient()

        assert client.flows is not None
        assert hasattr(client.flows, "list")
        assert hasattr(client.flows, "get")
        assert hasattr(client.flows, "create")
        assert hasattr(client.flows, "update")
        assert hasattr(client.flows, "delete")
        assert hasattr(client.flows, "execute")
        assert hasattr(client.flows, "activate")
        assert hasattr(client.flows, "deactivate")


class TestAnalyticsResource:
    """Tests for AnalyticsResource"""

    def test_analytics_resource_exists(self):
        """Should have analytics resource"""
        client = LinktorClient()

        assert client.analytics is not None
        assert hasattr(client.analytics, "get_dashboard")
        assert hasattr(client.analytics, "get_conversation_metrics")
        assert hasattr(client.analytics, "get_message_metrics")
        assert hasattr(client.analytics, "get_realtime")
