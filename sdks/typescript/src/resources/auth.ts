/**
 * Auth Resource
 */

import type { HttpClient } from '../utils/http';
import type {
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  User,
  UpdateUserRequest,
} from '../types/auth';

export class AuthResource {
  constructor(private http: HttpClient) {}

  /**
   * Login with email and password
   */
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await this.http.post<LoginResponse>('/auth/login', credentials);
    // Update http client with new token
    this.http.setAccessToken(response.accessToken);
    return response;
  }

  /**
   * Logout current session
   */
  async logout(): Promise<void> {
    await this.http.post<void>('/auth/logout');
  }

  /**
   * Refresh access token
   */
  async refreshToken(request: RefreshTokenRequest): Promise<RefreshTokenResponse> {
    const response = await this.http.post<RefreshTokenResponse>('/auth/refresh', request);
    // Update http client with new token
    this.http.setAccessToken(response.accessToken);
    return response;
  }

  /**
   * Get current user
   */
  async getCurrentUser(): Promise<User> {
    return this.http.get<User>('/auth/me');
  }

  /**
   * Update current user profile
   */
  async updateProfile(data: UpdateUserRequest): Promise<User> {
    return this.http.patch<User>('/auth/me', data);
  }

  /**
   * Change password
   */
  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await this.http.post<void>('/auth/change-password', {
      currentPassword,
      newPassword,
    });
  }
}
