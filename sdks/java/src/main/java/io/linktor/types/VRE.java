package io.linktor.types;

import com.google.gson.annotations.SerializedName;

import java.util.Map;

/**
 * VRE (Visual Response Engine) types
 */
public class VRE {

    /**
     * Output format for rendered images
     */
    public enum OutputFormat {
        @SerializedName("png") PNG,
        @SerializedName("webp") WEBP,
        @SerializedName("jpeg") JPEG
    }

    /**
     * Channel type for VRE rendering
     */
    public enum ChannelType {
        @SerializedName("whatsapp") WHATSAPP,
        @SerializedName("telegram") TELEGRAM,
        @SerializedName("web") WEB,
        @SerializedName("email") EMAIL
    }

    /**
     * Available template types
     */
    public enum TemplateType {
        @SerializedName("menu_opcoes") MENU_OPCOES,
        @SerializedName("card_produto") CARD_PRODUTO,
        @SerializedName("status_pedido") STATUS_PEDIDO,
        @SerializedName("lista_produtos") LISTA_PRODUTOS,
        @SerializedName("confirmacao") CONFIRMACAO,
        @SerializedName("cobranca_pix") COBRANCA_PIX
    }

    /**
     * Order status for status_pedido template
     */
    public enum OrderStatus {
        @SerializedName("recebido") RECEBIDO,
        @SerializedName("separacao") SEPARACAO,
        @SerializedName("faturado") FATURADO,
        @SerializedName("transporte") TRANSPORTE,
        @SerializedName("entregue") ENTREGUE
    }

    /**
     * Render request
     */
    public static class RenderRequest {
        @SerializedName("tenant_id")
        private String tenantId;

        @SerializedName("template_id")
        private String templateId;

        private Map<String, Object> data;

        private String channel;

        private String format;

        private Integer width;

        private Integer quality;

        private Double scale;

        // Getters
        public String getTenantId() { return tenantId; }
        public String getTemplateId() { return templateId; }
        public Map<String, Object> getData() { return data; }
        public String getChannel() { return channel; }
        public String getFormat() { return format; }
        public Integer getWidth() { return width; }
        public Integer getQuality() { return quality; }
        public Double getScale() { return scale; }

        // Builder
        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final RenderRequest request = new RenderRequest();

            public Builder tenantId(String tenantId) {
                request.tenantId = tenantId;
                return this;
            }

            public Builder templateId(String templateId) {
                request.templateId = templateId;
                return this;
            }

            public Builder templateId(TemplateType templateId) {
                request.templateId = templateId.name().toLowerCase();
                return this;
            }

            public Builder data(Map<String, Object> data) {
                request.data = data;
                return this;
            }

            public Builder channel(String channel) {
                request.channel = channel;
                return this;
            }

            public Builder channel(ChannelType channel) {
                request.channel = channel.name().toLowerCase();
                return this;
            }

            public Builder format(String format) {
                request.format = format;
                return this;
            }

            public Builder format(OutputFormat format) {
                request.format = format.name().toLowerCase();
                return this;
            }

            public Builder width(Integer width) {
                request.width = width;
                return this;
            }

            public Builder quality(Integer quality) {
                request.quality = quality;
                return this;
            }

            public Builder scale(Double scale) {
                request.scale = scale;
                return this;
            }

            public RenderRequest build() { return request; }
        }
    }

    /**
     * Render response
     */
    public static class RenderResponse {
        @SerializedName("image_base64")
        private String imageBase64;

        private String caption;

        private int width;

        private int height;

        private String format;

        @SerializedName("render_time_ms")
        private int renderTimeMs;

        @SerializedName("size_bytes")
        private Long sizeBytes;

        @SerializedName("cache_hit")
        private Boolean cacheHit;

        // Getters
        public String getImageBase64() { return imageBase64; }
        public String getCaption() { return caption; }
        public int getWidth() { return width; }
        public int getHeight() { return height; }
        public String getFormat() { return format; }
        public int getRenderTimeMs() { return renderTimeMs; }
        public Long getSizeBytes() { return sizeBytes; }
        public Boolean getCacheHit() { return cacheHit; }
    }

    /**
     * Render and send request
     */
    public static class RenderAndSendRequest {
        @SerializedName("conversation_id")
        private String conversationId;

        @SerializedName("template_id")
        private String templateId;

        private Map<String, Object> data;

        private String caption;

        @SerializedName("follow_up_text")
        private String followUpText;

        // Getters
        public String getConversationId() { return conversationId; }
        public String getTemplateId() { return templateId; }
        public Map<String, Object> getData() { return data; }
        public String getCaption() { return caption; }
        public String getFollowUpText() { return followUpText; }

        // Builder
        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final RenderAndSendRequest request = new RenderAndSendRequest();

            public Builder conversationId(String conversationId) {
                request.conversationId = conversationId;
                return this;
            }

            public Builder templateId(String templateId) {
                request.templateId = templateId;
                return this;
            }

            public Builder templateId(TemplateType templateId) {
                request.templateId = templateId.name().toLowerCase();
                return this;
            }

            public Builder data(Map<String, Object> data) {
                request.data = data;
                return this;
            }

            public Builder caption(String caption) {
                request.caption = caption;
                return this;
            }

            public Builder followUpText(String followUpText) {
                request.followUpText = followUpText;
                return this;
            }

            public RenderAndSendRequest build() { return request; }
        }
    }

    /**
     * Render and send response
     */
    public static class RenderAndSendResponse {
        @SerializedName("message_id")
        private String messageId;

        @SerializedName("image_url")
        private String imageUrl;

        private String caption;

        // Getters
        public String getMessageId() { return messageId; }
        public String getImageUrl() { return imageUrl; }
        public String getCaption() { return caption; }
    }

    /**
     * Template definition
     */
    public static class Template {
        private String id;
        private String name;
        private String description;
        private Map<String, Object> schema;

        // Getters
        public String getId() { return id; }
        public String getName() { return name; }
        public String getDescription() { return description; }
        public Map<String, Object> getSchema() { return schema; }
    }

    /**
     * List templates response
     */
    public static class ListTemplatesResponse {
        private java.util.List<Template> templates;

        public java.util.List<Template> getTemplates() { return templates; }
    }

    /**
     * Preview request
     */
    public static class PreviewRequest {
        private Map<String, Object> data;

        public PreviewRequest() {}

        public PreviewRequest(Map<String, Object> data) {
            this.data = data;
        }

        public Map<String, Object> getData() { return data; }
        public void setData(Map<String, Object> data) { this.data = data; }
    }

    /**
     * Preview response
     */
    public static class PreviewResponse {
        @SerializedName("image_base64")
        private String imageBase64;

        private int width;
        private int height;

        // Getters
        public String getImageBase64() { return imageBase64; }
        public int getWidth() { return width; }
        public int getHeight() { return height; }
    }
}
