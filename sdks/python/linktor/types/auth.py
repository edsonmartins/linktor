"""
Authentication types
"""

from datetime import datetime
from enum import Enum
from typing import Optional
from pydantic import BaseModel, Field


class UserRole(str, Enum):
    """User roles"""

    ADMIN = "admin"
    AGENT = "agent"
    SUPERVISOR = "supervisor"
    VIEWER = "viewer"


class User(BaseModel):
    """User model"""

    id: str
    email: str
    name: str
    role: UserRole
    tenant_id: str = Field(alias="tenantId")
    avatar_url: Optional[str] = Field(None, alias="avatarUrl")
    phone: Optional[str] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class LoginRequest(BaseModel):
    """Login request"""

    email: str
    password: str


class LoginResponse(BaseModel):
    """Login response"""

    access_token: str = Field(alias="accessToken")
    refresh_token: str = Field(alias="refreshToken")
    expires_in: int = Field(alias="expiresIn")
    token_type: str = Field(alias="tokenType")
    user: User

    class Config:
        populate_by_name = True


class RefreshTokenRequest(BaseModel):
    """Refresh token request"""

    refresh_token: str = Field(alias="refreshToken")

    class Config:
        populate_by_name = True


class RefreshTokenResponse(BaseModel):
    """Refresh token response"""

    access_token: str = Field(alias="accessToken")
    refresh_token: str = Field(alias="refreshToken")
    expires_in: int = Field(alias="expiresIn")

    class Config:
        populate_by_name = True


class CreateUserRequest(BaseModel):
    """Create user request"""

    email: str
    password: str
    name: str
    role: UserRole
    phone: Optional[str] = None


class UpdateUserRequest(BaseModel):
    """Update user request"""

    name: Optional[str] = None
    phone: Optional[str] = None
    avatar_url: Optional[str] = Field(None, alias="avatarUrl")

    class Config:
        populate_by_name = True


__all__ = [
    "UserRole",
    "User",
    "LoginRequest",
    "LoginResponse",
    "RefreshTokenRequest",
    "RefreshTokenResponse",
    "CreateUserRequest",
    "UpdateUserRequest",
]
