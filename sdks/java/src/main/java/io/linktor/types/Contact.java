package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class Contact {

    public static class ContactModel {
        private String id;
        private String tenantId;
        private String name;
        private String email;
        private String phone;
        private String avatar;
        private Map<String, String> identifiers;
        private Map<String, Object> customFields;
        private List<String> tags;
        private Map<String, Object> metadata;
        private Instant lastSeenAt;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }

        public String getPhone() { return phone; }
        public void setPhone(String phone) { this.phone = phone; }

        public String getAvatar() { return avatar; }
        public void setAvatar(String avatar) { this.avatar = avatar; }

        public Map<String, String> getIdentifiers() { return identifiers; }
        public void setIdentifiers(Map<String, String> identifiers) { this.identifiers = identifiers; }

        public Map<String, Object> getCustomFields() { return customFields; }
        public void setCustomFields(Map<String, Object> customFields) { this.customFields = customFields; }

        public List<String> getTags() { return tags; }
        public void setTags(List<String> tags) { this.tags = tags; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getLastSeenAt() { return lastSeenAt; }
        public void setLastSeenAt(Instant lastSeenAt) { this.lastSeenAt = lastSeenAt; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class CreateContactInput {
        private String name;
        private String email;
        private String phone;
        private String avatar;
        private Map<String, String> identifiers;
        private Map<String, Object> customFields;
        private List<String> tags;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }

        public String getPhone() { return phone; }
        public void setPhone(String phone) { this.phone = phone; }

        public String getAvatar() { return avatar; }
        public void setAvatar(String avatar) { this.avatar = avatar; }

        public Map<String, String> getIdentifiers() { return identifiers; }
        public void setIdentifiers(Map<String, String> identifiers) { this.identifiers = identifiers; }

        public Map<String, Object> getCustomFields() { return customFields; }
        public void setCustomFields(Map<String, Object> customFields) { this.customFields = customFields; }

        public List<String> getTags() { return tags; }
        public void setTags(List<String> tags) { this.tags = tags; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CreateContactInput input = new CreateContactInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder email(String email) { input.email = email; return this; }
            public Builder phone(String phone) { input.phone = phone; return this; }
            public Builder avatar(String avatar) { input.avatar = avatar; return this; }
            public Builder identifiers(Map<String, String> identifiers) { input.identifiers = identifiers; return this; }
            public Builder customFields(Map<String, Object> customFields) { input.customFields = customFields; return this; }
            public Builder tags(List<String> tags) { input.tags = tags; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public CreateContactInput build() { return input; }
        }
    }

    public static class UpdateContactInput {
        private String name;
        private String email;
        private String phone;
        private String avatar;
        private Map<String, String> identifiers;
        private Map<String, Object> customFields;
        private List<String> tags;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }

        public String getPhone() { return phone; }
        public void setPhone(String phone) { this.phone = phone; }

        public String getAvatar() { return avatar; }
        public void setAvatar(String avatar) { this.avatar = avatar; }

        public Map<String, String> getIdentifiers() { return identifiers; }
        public void setIdentifiers(Map<String, String> identifiers) { this.identifiers = identifiers; }

        public Map<String, Object> getCustomFields() { return customFields; }
        public void setCustomFields(Map<String, Object> customFields) { this.customFields = customFields; }

        public List<String> getTags() { return tags; }
        public void setTags(List<String> tags) { this.tags = tags; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final UpdateContactInput input = new UpdateContactInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder email(String email) { input.email = email; return this; }
            public Builder phone(String phone) { input.phone = phone; return this; }
            public Builder avatar(String avatar) { input.avatar = avatar; return this; }
            public Builder identifiers(Map<String, String> identifiers) { input.identifiers = identifiers; return this; }
            public Builder customFields(Map<String, Object> customFields) { input.customFields = customFields; return this; }
            public Builder tags(List<String> tags) { input.tags = tags; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public UpdateContactInput build() { return input; }
        }
    }

    public static class ListContactsParams extends Common.PaginationParams {
        private String tag;
        private String search;
        private String email;
        private String phone;

        public String getTag() { return tag; }
        public void setTag(String tag) { this.tag = tag; }

        public String getSearch() { return search; }
        public void setSearch(String search) { this.search = search; }

        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }

        public String getPhone() { return phone; }
        public void setPhone(String phone) { this.phone = phone; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final ListContactsParams params = new ListContactsParams();

            public Builder tag(String tag) { params.tag = tag; return this; }
            public Builder search(String search) { params.search = search; return this; }
            public Builder email(String email) { params.email = email; return this; }
            public Builder phone(String phone) { params.phone = phone; return this; }
            public Builder limit(Integer limit) { params.setLimit(limit); return this; }
            public Builder page(Integer page) { params.setPage(page); return this; }
            public ListContactsParams build() { return params; }
        }
    }

    public static class MergeContactsInput {
        private String primaryContactId;
        private List<String> contactIdsToMerge;

        public String getPrimaryContactId() { return primaryContactId; }
        public void setPrimaryContactId(String primaryContactId) { this.primaryContactId = primaryContactId; }

        public List<String> getContactIdsToMerge() { return contactIdsToMerge; }
        public void setContactIdsToMerge(List<String> contactIdsToMerge) { this.contactIdsToMerge = contactIdsToMerge; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final MergeContactsInput input = new MergeContactsInput();

            public Builder primaryContactId(String id) { input.primaryContactId = id; return this; }
            public Builder contactIdsToMerge(List<String> ids) { input.contactIdsToMerge = ids; return this; }
            public MergeContactsInput build() { return input; }
        }
    }
}
