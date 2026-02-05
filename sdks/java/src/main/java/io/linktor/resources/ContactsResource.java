package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.Common;
import io.linktor.types.Contact;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.Map;

public class ContactsResource {
    private final HttpClient http;

    public ContactsResource(HttpClient http) {
        this.http = http;
    }

    /**
     * List contacts with optional filters
     */
    public Common.PaginatedResponse<Contact.ContactModel> list(Contact.ListContactsParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getTag() != null) queryParams.put("tag", params.getTag());
            if (params.getSearch() != null) queryParams.put("search", params.getSearch());
            if (params.getEmail() != null) queryParams.put("email", params.getEmail());
            if (params.getPhone() != null) queryParams.put("phone", params.getPhone());
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
            if (params.getCursor() != null) queryParams.put("cursor", params.getCursor());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Contact.ContactModel>>(){}.getType();
        return http.get("/contacts", queryParams, responseType);
    }

    /**
     * List all contacts (no filters)
     */
    public Common.PaginatedResponse<Contact.ContactModel> list() throws LinktorException {
        return list(null);
    }

    /**
     * Get a contact by ID
     */
    public Contact.ContactModel get(String contactId) throws LinktorException {
        return http.get("/contacts/" + contactId, Contact.ContactModel.class);
    }

    /**
     * Create a new contact
     */
    public Contact.ContactModel create(Contact.CreateContactInput input) throws LinktorException {
        return http.post("/contacts", input, Contact.ContactModel.class);
    }

    /**
     * Update a contact
     */
    public Contact.ContactModel update(String contactId, Contact.UpdateContactInput input) throws LinktorException {
        return http.patch("/contacts/" + contactId, input, Contact.ContactModel.class);
    }

    /**
     * Delete a contact
     */
    public void delete(String contactId) throws LinktorException {
        http.delete("/contacts/" + contactId);
    }

    /**
     * Find contact by email
     */
    public Contact.ContactModel findByEmail(String email) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        queryParams.put("email", email);
        Type responseType = new TypeToken<Common.PaginatedResponse<Contact.ContactModel>>(){}.getType();
        Common.PaginatedResponse<Contact.ContactModel> response = http.get("/contacts", queryParams, responseType);
        if (response.getData() != null && !response.getData().isEmpty()) {
            return response.getData().get(0);
        }
        return null;
    }

    /**
     * Find contact by phone
     */
    public Contact.ContactModel findByPhone(String phone) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        queryParams.put("phone", phone);
        Type responseType = new TypeToken<Common.PaginatedResponse<Contact.ContactModel>>(){}.getType();
        Common.PaginatedResponse<Contact.ContactModel> response = http.get("/contacts", queryParams, responseType);
        if (response.getData() != null && !response.getData().isEmpty()) {
            return response.getData().get(0);
        }
        return null;
    }

    /**
     * Merge contacts
     */
    public Contact.ContactModel merge(Contact.MergeContactsInput input) throws LinktorException {
        return http.post("/contacts/merge", input, Contact.ContactModel.class);
    }

    /**
     * Get or create contact by identifier
     */
    public Contact.ContactModel getOrCreate(Contact.CreateContactInput input) throws LinktorException {
        // Try to find by email first
        if (input.getEmail() != null) {
            Contact.ContactModel existing = findByEmail(input.getEmail());
            if (existing != null) return existing;
        }

        // Try to find by phone
        if (input.getPhone() != null) {
            Contact.ContactModel existing = findByPhone(input.getPhone());
            if (existing != null) return existing;
        }

        // Create new contact
        return create(input);
    }
}
