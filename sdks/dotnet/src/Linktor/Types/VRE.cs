using System.Text.Json.Serialization;

namespace Linktor.Types;

/// <summary>
/// VRE render response
/// </summary>
public class VRERenderResponse
{
    [JsonPropertyName("image_base64")]
    public string ImageBase64 { get; set; } = string.Empty;

    [JsonPropertyName("caption")]
    public string Caption { get; set; } = string.Empty;

    [JsonPropertyName("width")]
    public int Width { get; set; }

    [JsonPropertyName("height")]
    public int Height { get; set; }

    [JsonPropertyName("format")]
    public string Format { get; set; } = string.Empty;

    [JsonPropertyName("render_time_ms")]
    public int RenderTimeMs { get; set; }

    [JsonPropertyName("size_bytes")]
    public long? SizeBytes { get; set; }

    [JsonPropertyName("cache_hit")]
    public bool? CacheHit { get; set; }
}

/// <summary>
/// VRE render and send response
/// </summary>
public class VRERenderAndSendResponse
{
    [JsonPropertyName("message_id")]
    public string MessageId { get; set; } = string.Empty;

    [JsonPropertyName("image_url")]
    public string ImageUrl { get; set; } = string.Empty;

    [JsonPropertyName("caption")]
    public string Caption { get; set; } = string.Empty;
}

/// <summary>
/// VRE template definition
/// </summary>
public class VRETemplate
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("description")]
    public string Description { get; set; } = string.Empty;

    [JsonPropertyName("schema")]
    public Dictionary<string, object>? Schema { get; set; }
}

/// <summary>
/// VRE list templates response
/// </summary>
public class VREListTemplatesResponse
{
    [JsonPropertyName("templates")]
    public List<VRETemplate> Templates { get; set; } = new();
}

/// <summary>
/// VRE preview response
/// </summary>
public class VREPreviewResponse
{
    [JsonPropertyName("image_base64")]
    public string ImageBase64 { get; set; } = string.Empty;

    [JsonPropertyName("width")]
    public int Width { get; set; }

    [JsonPropertyName("height")]
    public int Height { get; set; }
}
