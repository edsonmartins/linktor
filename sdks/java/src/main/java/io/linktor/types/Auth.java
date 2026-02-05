package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class Auth {

    public enum UserRole {
        @SerializedName("admin") ADMIN,
        @SerializedName("manager") MANAGER,
        @SerializedName("agent") AGENT,
        @SerializedName("viewer") VIEWER
    }

    public enum UserStatus {
        @SerializedName("active") ACTIVE,
        @SerializedName("inactive") INACTIVE,
        @SerializedName("pending") PENDING,
        @SerializedName("suspended") SUSPENDED
    }

    public static class User {
        private String id;
        private String tenantId;
        private String email;
        private String name;
        private String avatar;
        private UserRole role;
        private UserStatus status;
        private Map<String, Object> preferences;
        private Map<String, Object> metadata;
        private Instant lastLoginAt;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getAvatar() { return avatar; }
        public void setAvatar(String avatar) { this.avatar = avatar; }

        public UserRole getRole() { return role; }
        public void setRole(UserRole role) { this.role = role; }

        public UserStatus getStatus() { return status; }
        public void setStatus(UserStatus status) { this.status = status; }

        public Map<String, Object> getPreferences() { return preferences; }
        public void setPreferences(Map<String, Object> preferences) { this.preferences = preferences; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getLastLoginAt() { return lastLoginAt; }
        public void setLastLoginAt(Instant lastLoginAt) { this.lastLoginAt = lastLoginAt; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class Tenant {
        private String id;
        private String name;
        private String slug;
        private String plan;
        private TenantSettings settings;
        private Map<String, Object> metadata;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getSlug() { return slug; }
        public void setSlug(String slug) { this.slug = slug; }

        public String getPlan() { return plan; }
        public void setPlan(String plan) { this.plan = plan; }

        public TenantSettings getSettings() { return settings; }
        public void setSettings(TenantSettings settings) { this.settings = settings; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class TenantSettings {
        private String timezone;
        private String language;
        private String dateFormat;
        private BusinessHours businessHours;
        private NotificationSettings notifications;

        public String getTimezone() { return timezone; }
        public void setTimezone(String timezone) { this.timezone = timezone; }

        public String getLanguage() { return language; }
        public void setLanguage(String language) { this.language = language; }

        public String getDateFormat() { return dateFormat; }
        public void setDateFormat(String dateFormat) { this.dateFormat = dateFormat; }

        public BusinessHours getBusinessHours() { return businessHours; }
        public void setBusinessHours(BusinessHours businessHours) { this.businessHours = businessHours; }

        public NotificationSettings getNotifications() { return notifications; }
        public void setNotifications(NotificationSettings notifications) { this.notifications = notifications; }
    }

    public static class BusinessHours {
        private boolean enabled;
        private String timezone;
        private Map<String, DaySchedule> schedule;

        public boolean isEnabled() { return enabled; }
        public void setEnabled(boolean enabled) { this.enabled = enabled; }

        public String getTimezone() { return timezone; }
        public void setTimezone(String timezone) { this.timezone = timezone; }

        public Map<String, DaySchedule> getSchedule() { return schedule; }
        public void setSchedule(Map<String, DaySchedule> schedule) { this.schedule = schedule; }
    }

    public static class DaySchedule {
        private boolean enabled;
        private String start;
        private String end;

        public boolean isEnabled() { return enabled; }
        public void setEnabled(boolean enabled) { this.enabled = enabled; }

        public String getStart() { return start; }
        public void setStart(String start) { this.start = start; }

        public String getEnd() { return end; }
        public void setEnd(String end) { this.end = end; }
    }

    public static class NotificationSettings {
        private boolean email;
        private boolean push;
        private boolean sound;

        public boolean isEmail() { return email; }
        public void setEmail(boolean email) { this.email = email; }

        public boolean isPush() { return push; }
        public void setPush(boolean push) { this.push = push; }

        public boolean isSound() { return sound; }
        public void setSound(boolean sound) { this.sound = sound; }
    }

    public static class LoginInput {
        private String email;
        private String password;

        public LoginInput() {}

        public LoginInput(String email, String password) {
            this.email = email;
            this.password = password;
        }

        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }

        public String getPassword() { return password; }
        public void setPassword(String password) { this.password = password; }
    }

    public static class LoginResponse {
        private User user;
        private Tenant tenant;
        private String accessToken;
        private String refreshToken;
        private long expiresIn;

        public User getUser() { return user; }
        public void setUser(User user) { this.user = user; }

        public Tenant getTenant() { return tenant; }
        public void setTenant(Tenant tenant) { this.tenant = tenant; }

        public String getAccessToken() { return accessToken; }
        public void setAccessToken(String accessToken) { this.accessToken = accessToken; }

        public String getRefreshToken() { return refreshToken; }
        public void setRefreshToken(String refreshToken) { this.refreshToken = refreshToken; }

        public long getExpiresIn() { return expiresIn; }
        public void setExpiresIn(long expiresIn) { this.expiresIn = expiresIn; }
    }

    public static class RefreshTokenInput {
        private String refreshToken;

        public RefreshTokenInput() {}

        public RefreshTokenInput(String refreshToken) {
            this.refreshToken = refreshToken;
        }

        public String getRefreshToken() { return refreshToken; }
        public void setRefreshToken(String refreshToken) { this.refreshToken = refreshToken; }
    }

    public static class RefreshTokenResponse {
        private String accessToken;
        private String refreshToken;
        private long expiresIn;

        public String getAccessToken() { return accessToken; }
        public void setAccessToken(String accessToken) { this.accessToken = accessToken; }

        public String getRefreshToken() { return refreshToken; }
        public void setRefreshToken(String refreshToken) { this.refreshToken = refreshToken; }

        public long getExpiresIn() { return expiresIn; }
        public void setExpiresIn(long expiresIn) { this.expiresIn = expiresIn; }
    }
}
