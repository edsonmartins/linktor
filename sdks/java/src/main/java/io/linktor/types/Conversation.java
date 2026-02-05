package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class Conversation {

    public enum ConversationStatus {
        @SerializedName("open") OPEN,
        @SerializedName("pending") PENDING,
        @SerializedName("resolved") RESOLVED,
        @SerializedName("closed") CLOSED
    }

    public enum ConversationPriority {
        @SerializedName("low") LOW,
        @SerializedName("medium") MEDIUM,
        @SerializedName("high") HIGH,
        @SerializedName("urgent") URGENT
    }

    public enum MessageType {
        @SerializedName("text") TEXT,
        @SerializedName("image") IMAGE,
        @SerializedName("video") VIDEO,
        @SerializedName("audio") AUDIO,
        @SerializedName("document") DOCUMENT,
        @SerializedName("location") LOCATION,
        @SerializedName("contact") CONTACT,
        @SerializedName("sticker") STICKER,
        @SerializedName("template") TEMPLATE,
        @SerializedName("interactive") INTERACTIVE,
        @SerializedName("system") SYSTEM
    }

    public enum MessageStatus {
        @SerializedName("pending") PENDING,
        @SerializedName("sent") SENT,
        @SerializedName("delivered") DELIVERED,
        @SerializedName("read") READ,
        @SerializedName("failed") FAILED
    }

    public enum MessageDirection {
        @SerializedName("inbound") INBOUND,
        @SerializedName("outbound") OUTBOUND
    }

    public static class ConversationModel {
        private String id;
        private String tenantId;
        private String channelId;
        private String contactId;
        private String assignedAgentId;
        private String botId;
        private ConversationStatus status;
        private ConversationPriority priority;
        private String subject;
        private Message lastMessage;
        private int unreadCount;
        private List<String> tags;
        private Map<String, Object> metadata;
        private Instant firstMessageAt;
        private Instant lastMessageAt;
        private Instant resolvedAt;
        private Instant createdAt;
        private Instant updatedAt;

        // Relations
        private Contact.ContactModel contact;
        private Channel.ChannelModel channel;
        private Auth.User assignedAgent;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public String getChannelId() { return channelId; }
        public void setChannelId(String channelId) { this.channelId = channelId; }

        public String getContactId() { return contactId; }
        public void setContactId(String contactId) { this.contactId = contactId; }

        public String getAssignedAgentId() { return assignedAgentId; }
        public void setAssignedAgentId(String assignedAgentId) { this.assignedAgentId = assignedAgentId; }

        public String getBotId() { return botId; }
        public void setBotId(String botId) { this.botId = botId; }

        public ConversationStatus getStatus() { return status; }
        public void setStatus(ConversationStatus status) { this.status = status; }

        public ConversationPriority getPriority() { return priority; }
        public void setPriority(ConversationPriority priority) { this.priority = priority; }

        public String getSubject() { return subject; }
        public void setSubject(String subject) { this.subject = subject; }

        public Message getLastMessage() { return lastMessage; }
        public void setLastMessage(Message lastMessage) { this.lastMessage = lastMessage; }

        public int getUnreadCount() { return unreadCount; }
        public void setUnreadCount(int unreadCount) { this.unreadCount = unreadCount; }

        public List<String> getTags() { return tags; }
        public void setTags(List<String> tags) { this.tags = tags; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getFirstMessageAt() { return firstMessageAt; }
        public void setFirstMessageAt(Instant firstMessageAt) { this.firstMessageAt = firstMessageAt; }

        public Instant getLastMessageAt() { return lastMessageAt; }
        public void setLastMessageAt(Instant lastMessageAt) { this.lastMessageAt = lastMessageAt; }

        public Instant getResolvedAt() { return resolvedAt; }
        public void setResolvedAt(Instant resolvedAt) { this.resolvedAt = resolvedAt; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }

        public Contact.ContactModel getContact() { return contact; }
        public void setContact(Contact.ContactModel contact) { this.contact = contact; }

        public Channel.ChannelModel getChannel() { return channel; }
        public void setChannel(Channel.ChannelModel channel) { this.channel = channel; }

        public Auth.User getAssignedAgent() { return assignedAgent; }
        public void setAssignedAgent(Auth.User assignedAgent) { this.assignedAgent = assignedAgent; }
    }

    public static class Message {
        private String id;
        private String conversationId;
        private MessageType type;
        private MessageDirection direction;
        private MessageStatus status;
        private String text;
        private MediaContent media;
        private LocationContent location;
        private ContactContent contactContent;
        private TemplateContent template;
        private InteractiveContent interactive;
        private String senderId;
        private String senderType;
        private String externalId;
        private Map<String, Object> metadata;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getConversationId() { return conversationId; }
        public void setConversationId(String conversationId) { this.conversationId = conversationId; }

        public MessageType getType() { return type; }
        public void setType(MessageType type) { this.type = type; }

        public MessageDirection getDirection() { return direction; }
        public void setDirection(MessageDirection direction) { this.direction = direction; }

        public MessageStatus getStatus() { return status; }
        public void setStatus(MessageStatus status) { this.status = status; }

        public String getText() { return text; }
        public void setText(String text) { this.text = text; }

        public MediaContent getMedia() { return media; }
        public void setMedia(MediaContent media) { this.media = media; }

        public LocationContent getLocation() { return location; }
        public void setLocation(LocationContent location) { this.location = location; }

        public ContactContent getContactContent() { return contactContent; }
        public void setContactContent(ContactContent contactContent) { this.contactContent = contactContent; }

        public TemplateContent getTemplate() { return template; }
        public void setTemplate(TemplateContent template) { this.template = template; }

        public InteractiveContent getInteractive() { return interactive; }
        public void setInteractive(InteractiveContent interactive) { this.interactive = interactive; }

        public String getSenderId() { return senderId; }
        public void setSenderId(String senderId) { this.senderId = senderId; }

        public String getSenderType() { return senderType; }
        public void setSenderType(String senderType) { this.senderType = senderType; }

        public String getExternalId() { return externalId; }
        public void setExternalId(String externalId) { this.externalId = externalId; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class MediaContent {
        private String url;
        private String mimeType;
        private String filename;
        private long size;
        private String caption;

        public String getUrl() { return url; }
        public void setUrl(String url) { this.url = url; }

        public String getMimeType() { return mimeType; }
        public void setMimeType(String mimeType) { this.mimeType = mimeType; }

        public String getFilename() { return filename; }
        public void setFilename(String filename) { this.filename = filename; }

        public long getSize() { return size; }
        public void setSize(long size) { this.size = size; }

        public String getCaption() { return caption; }
        public void setCaption(String caption) { this.caption = caption; }
    }

    public static class LocationContent {
        private double latitude;
        private double longitude;
        private String name;
        private String address;

        public double getLatitude() { return latitude; }
        public void setLatitude(double latitude) { this.latitude = latitude; }

        public double getLongitude() { return longitude; }
        public void setLongitude(double longitude) { this.longitude = longitude; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getAddress() { return address; }
        public void setAddress(String address) { this.address = address; }
    }

    public static class ContactContent {
        private String name;
        private List<PhoneNumber> phones;
        private List<String> emails;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public List<PhoneNumber> getPhones() { return phones; }
        public void setPhones(List<PhoneNumber> phones) { this.phones = phones; }

        public List<String> getEmails() { return emails; }
        public void setEmails(List<String> emails) { this.emails = emails; }
    }

    public static class PhoneNumber {
        private String type;
        private String number;

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public String getNumber() { return number; }
        public void setNumber(String number) { this.number = number; }
    }

    public static class TemplateContent {
        private String name;
        private String language;
        private List<TemplateComponent> components;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getLanguage() { return language; }
        public void setLanguage(String language) { this.language = language; }

        public List<TemplateComponent> getComponents() { return components; }
        public void setComponents(List<TemplateComponent> components) { this.components = components; }
    }

    public static class TemplateComponent {
        private String type;
        private List<TemplateParameter> parameters;

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public List<TemplateParameter> getParameters() { return parameters; }
        public void setParameters(List<TemplateParameter> parameters) { this.parameters = parameters; }
    }

    public static class TemplateParameter {
        private String type;
        private String text;
        private MediaContent image;
        private MediaContent document;
        private MediaContent video;

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public String getText() { return text; }
        public void setText(String text) { this.text = text; }

        public MediaContent getImage() { return image; }
        public void setImage(MediaContent image) { this.image = image; }

        public MediaContent getDocument() { return document; }
        public void setDocument(MediaContent document) { this.document = document; }

        public MediaContent getVideo() { return video; }
        public void setVideo(MediaContent video) { this.video = video; }
    }

    public static class InteractiveContent {
        private String type;
        private InteractiveHeader header;
        private InteractiveBody body;
        private InteractiveFooter footer;
        private InteractiveAction action;

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public InteractiveHeader getHeader() { return header; }
        public void setHeader(InteractiveHeader header) { this.header = header; }

        public InteractiveBody getBody() { return body; }
        public void setBody(InteractiveBody body) { this.body = body; }

        public InteractiveFooter getFooter() { return footer; }
        public void setFooter(InteractiveFooter footer) { this.footer = footer; }

        public InteractiveAction getAction() { return action; }
        public void setAction(InteractiveAction action) { this.action = action; }
    }

    public static class InteractiveHeader {
        private String type;
        private String text;
        private MediaContent image;
        private MediaContent video;
        private MediaContent document;

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public String getText() { return text; }
        public void setText(String text) { this.text = text; }

        public MediaContent getImage() { return image; }
        public void setImage(MediaContent image) { this.image = image; }

        public MediaContent getVideo() { return video; }
        public void setVideo(MediaContent video) { this.video = video; }

        public MediaContent getDocument() { return document; }
        public void setDocument(MediaContent document) { this.document = document; }
    }

    public static class InteractiveBody {
        private String text;

        public String getText() { return text; }
        public void setText(String text) { this.text = text; }
    }

    public static class InteractiveFooter {
        private String text;

        public String getText() { return text; }
        public void setText(String text) { this.text = text; }
    }

    public static class InteractiveAction {
        private List<Button> buttons;
        private List<Section> sections;

        public List<Button> getButtons() { return buttons; }
        public void setButtons(List<Button> buttons) { this.buttons = buttons; }

        public List<Section> getSections() { return sections; }
        public void setSections(List<Section> sections) { this.sections = sections; }
    }

    public static class Button {
        private String type;
        private String id;
        private String title;

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTitle() { return title; }
        public void setTitle(String title) { this.title = title; }
    }

    public static class Section {
        private String title;
        private List<SectionRow> rows;

        public String getTitle() { return title; }
        public void setTitle(String title) { this.title = title; }

        public List<SectionRow> getRows() { return rows; }
        public void setRows(List<SectionRow> rows) { this.rows = rows; }
    }

    public static class SectionRow {
        private String id;
        private String title;
        private String description;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTitle() { return title; }
        public void setTitle(String title) { this.title = title; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
    }

    // Input classes

    public static class ListConversationsParams extends Common.PaginationParams {
        private ConversationStatus status;
        private ConversationPriority priority;
        private String channelId;
        private String contactId;
        private String assignedAgentId;
        private String tag;
        private String search;

        public ConversationStatus getStatus() { return status; }
        public void setStatus(ConversationStatus status) { this.status = status; }

        public ConversationPriority getPriority() { return priority; }
        public void setPriority(ConversationPriority priority) { this.priority = priority; }

        public String getChannelId() { return channelId; }
        public void setChannelId(String channelId) { this.channelId = channelId; }

        public String getContactId() { return contactId; }
        public void setContactId(String contactId) { this.contactId = contactId; }

        public String getAssignedAgentId() { return assignedAgentId; }
        public void setAssignedAgentId(String assignedAgentId) { this.assignedAgentId = assignedAgentId; }

        public String getTag() { return tag; }
        public void setTag(String tag) { this.tag = tag; }

        public String getSearch() { return search; }
        public void setSearch(String search) { this.search = search; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final ListConversationsParams params = new ListConversationsParams();

            public Builder status(ConversationStatus status) { params.status = status; return this; }
            public Builder priority(ConversationPriority priority) { params.priority = priority; return this; }
            public Builder channelId(String channelId) { params.channelId = channelId; return this; }
            public Builder contactId(String contactId) { params.contactId = contactId; return this; }
            public Builder assignedAgentId(String assignedAgentId) { params.assignedAgentId = assignedAgentId; return this; }
            public Builder tag(String tag) { params.tag = tag; return this; }
            public Builder search(String search) { params.search = search; return this; }
            public Builder limit(Integer limit) { params.setLimit(limit); return this; }
            public Builder page(Integer page) { params.setPage(page); return this; }
            public ListConversationsParams build() { return params; }
        }
    }

    public static class SendMessageInput {
        private String text;
        private MessageType type;
        private MediaContent media;
        private LocationContent location;
        private ContactContent contact;
        private TemplateContent template;
        private InteractiveContent interactive;
        private Map<String, Object> metadata;

        public String getText() { return text; }
        public void setText(String text) { this.text = text; }

        public MessageType getType() { return type; }
        public void setType(MessageType type) { this.type = type; }

        public MediaContent getMedia() { return media; }
        public void setMedia(MediaContent media) { this.media = media; }

        public LocationContent getLocation() { return location; }
        public void setLocation(LocationContent location) { this.location = location; }

        public ContactContent getContact() { return contact; }
        public void setContact(ContactContent contact) { this.contact = contact; }

        public TemplateContent getTemplate() { return template; }
        public void setTemplate(TemplateContent template) { this.template = template; }

        public InteractiveContent getInteractive() { return interactive; }
        public void setInteractive(InteractiveContent interactive) { this.interactive = interactive; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final SendMessageInput input = new SendMessageInput();

            public Builder text(String text) { input.text = text; input.type = MessageType.TEXT; return this; }
            public Builder type(MessageType type) { input.type = type; return this; }
            public Builder media(MediaContent media) { input.media = media; return this; }
            public Builder location(LocationContent location) { input.location = location; return this; }
            public Builder contact(ContactContent contact) { input.contact = contact; return this; }
            public Builder template(TemplateContent template) { input.template = template; return this; }
            public Builder interactive(InteractiveContent interactive) { input.interactive = interactive; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public SendMessageInput build() { return input; }
        }
    }

    public static class UpdateConversationInput {
        private ConversationStatus status;
        private ConversationPriority priority;
        private String assignedAgentId;
        private List<String> tags;
        private Map<String, Object> metadata;

        public ConversationStatus getStatus() { return status; }
        public void setStatus(ConversationStatus status) { this.status = status; }

        public ConversationPriority getPriority() { return priority; }
        public void setPriority(ConversationPriority priority) { this.priority = priority; }

        public String getAssignedAgentId() { return assignedAgentId; }
        public void setAssignedAgentId(String assignedAgentId) { this.assignedAgentId = assignedAgentId; }

        public List<String> getTags() { return tags; }
        public void setTags(List<String> tags) { this.tags = tags; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final UpdateConversationInput input = new UpdateConversationInput();

            public Builder status(ConversationStatus status) { input.status = status; return this; }
            public Builder priority(ConversationPriority priority) { input.priority = priority; return this; }
            public Builder assignedAgentId(String agentId) { input.assignedAgentId = agentId; return this; }
            public Builder tags(List<String> tags) { input.tags = tags; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public UpdateConversationInput build() { return input; }
        }
    }
}
