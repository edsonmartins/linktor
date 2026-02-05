using System.Text.Json.Serialization;

namespace Linktor.Types;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum UserRole
{
    Admin,
    Manager,
    Agent,
    Viewer
}

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum UserStatus
{
    Active,
    Inactive,
    Pending,
    Suspended
}

public class User
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("tenantId")]
    public string TenantId { get; set; } = string.Empty;

    [JsonPropertyName("email")]
    public string Email { get; set; } = string.Empty;

    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("avatar")]
    public string? Avatar { get; set; }

    [JsonPropertyName("role")]
    public UserRole Role { get; set; }

    [JsonPropertyName("status")]
    public UserStatus Status { get; set; }

    [JsonPropertyName("preferences")]
    public Dictionary<string, object>? Preferences { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }

    [JsonPropertyName("lastLoginAt")]
    public DateTime? LastLoginAt { get; set; }

    [JsonPropertyName("createdAt")]
    public DateTime CreatedAt { get; set; }

    [JsonPropertyName("updatedAt")]
    public DateTime UpdatedAt { get; set; }
}

public class Tenant
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("slug")]
    public string Slug { get; set; } = string.Empty;

    [JsonPropertyName("plan")]
    public string? Plan { get; set; }

    [JsonPropertyName("settings")]
    public TenantSettings? Settings { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }

    [JsonPropertyName("createdAt")]
    public DateTime CreatedAt { get; set; }

    [JsonPropertyName("updatedAt")]
    public DateTime UpdatedAt { get; set; }
}

public class TenantSettings
{
    [JsonPropertyName("timezone")]
    public string? Timezone { get; set; }

    [JsonPropertyName("language")]
    public string? Language { get; set; }

    [JsonPropertyName("dateFormat")]
    public string? DateFormat { get; set; }

    [JsonPropertyName("businessHours")]
    public BusinessHours? BusinessHours { get; set; }

    [JsonPropertyName("notifications")]
    public NotificationSettings? Notifications { get; set; }
}

public class BusinessHours
{
    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }

    [JsonPropertyName("timezone")]
    public string? Timezone { get; set; }

    [JsonPropertyName("schedule")]
    public Dictionary<string, DaySchedule>? Schedule { get; set; }
}

public class DaySchedule
{
    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }

    [JsonPropertyName("start")]
    public string? Start { get; set; }

    [JsonPropertyName("end")]
    public string? End { get; set; }
}

public class NotificationSettings
{
    [JsonPropertyName("email")]
    public bool Email { get; set; }

    [JsonPropertyName("push")]
    public bool Push { get; set; }

    [JsonPropertyName("sound")]
    public bool Sound { get; set; }
}

public class LoginInput
{
    [JsonPropertyName("email")]
    public string Email { get; set; } = string.Empty;

    [JsonPropertyName("password")]
    public string Password { get; set; } = string.Empty;

    public LoginInput() { }

    public LoginInput(string email, string password)
    {
        Email = email;
        Password = password;
    }
}

public class LoginResponse
{
    [JsonPropertyName("user")]
    public User User { get; set; } = new();

    [JsonPropertyName("tenant")]
    public Tenant Tenant { get; set; } = new();

    [JsonPropertyName("accessToken")]
    public string AccessToken { get; set; } = string.Empty;

    [JsonPropertyName("refreshToken")]
    public string RefreshToken { get; set; } = string.Empty;

    [JsonPropertyName("expiresIn")]
    public long ExpiresIn { get; set; }
}

public class RefreshTokenInput
{
    [JsonPropertyName("refreshToken")]
    public string RefreshToken { get; set; } = string.Empty;
}

public class RefreshTokenResponse
{
    [JsonPropertyName("accessToken")]
    public string AccessToken { get; set; } = string.Empty;

    [JsonPropertyName("refreshToken")]
    public string RefreshToken { get; set; } = string.Empty;

    [JsonPropertyName("expiresIn")]
    public long ExpiresIn { get; set; }
}
