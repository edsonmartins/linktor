package io.linktor.resources;

import io.linktor.types.Auth;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

public class AuthResource {
    private final HttpClient http;

    public AuthResource(HttpClient http) {
        this.http = http;
    }

    /**
     * Login with email and password
     */
    public Auth.LoginResponse login(String email, String password) throws LinktorException {
        Auth.LoginInput input = new Auth.LoginInput(email, password);
        Auth.LoginResponse response = http.post("/auth/login", input, Auth.LoginResponse.class);

        // Update HTTP client with new access token
        if (response != null && response.getAccessToken() != null) {
            http.setAccessToken(response.getAccessToken());
        }

        return response;
    }

    /**
     * Login with LoginInput
     */
    public Auth.LoginResponse login(Auth.LoginInput input) throws LinktorException {
        Auth.LoginResponse response = http.post("/auth/login", input, Auth.LoginResponse.class);

        // Update HTTP client with new access token
        if (response != null && response.getAccessToken() != null) {
            http.setAccessToken(response.getAccessToken());
        }

        return response;
    }

    /**
     * Logout current user
     */
    public void logout() throws LinktorException {
        http.post("/auth/logout", null, Void.class);
        http.setAccessToken(null);
    }

    /**
     * Refresh access token
     */
    public Auth.RefreshTokenResponse refreshToken(String refreshToken) throws LinktorException {
        Auth.RefreshTokenInput input = new Auth.RefreshTokenInput(refreshToken);
        Auth.RefreshTokenResponse response = http.post("/auth/refresh", input, Auth.RefreshTokenResponse.class);

        // Update HTTP client with new access token
        if (response != null && response.getAccessToken() != null) {
            http.setAccessToken(response.getAccessToken());
        }

        return response;
    }

    /**
     * Get current authenticated user
     */
    public Auth.User getCurrentUser() throws LinktorException {
        return http.get("/auth/me", Auth.User.class);
    }

    /**
     * Get current tenant
     */
    public Auth.Tenant getCurrentTenant() throws LinktorException {
        return http.get("/auth/tenant", Auth.Tenant.class);
    }
}
