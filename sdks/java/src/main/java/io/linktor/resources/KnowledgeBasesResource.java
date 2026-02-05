package io.linktor.resources;

import com.google.gson.reflect.TypeToken;
import io.linktor.types.Common;
import io.linktor.types.Knowledge;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.Map;

public class KnowledgeBasesResource {
    private final HttpClient http;

    public KnowledgeBasesResource(HttpClient http) {
        this.http = http;
    }

    /**
     * List knowledge bases
     */
    public Common.PaginatedResponse<Knowledge.KnowledgeBase> list(Common.PaginationParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Knowledge.KnowledgeBase>>(){}.getType();
        return http.get("/knowledge-bases", queryParams, responseType);
    }

    /**
     * List all knowledge bases (no pagination)
     */
    public Common.PaginatedResponse<Knowledge.KnowledgeBase> list() throws LinktorException {
        return list(null);
    }

    /**
     * Get a knowledge base by ID
     */
    public Knowledge.KnowledgeBase get(String kbId) throws LinktorException {
        return http.get("/knowledge-bases/" + kbId, Knowledge.KnowledgeBase.class);
    }

    /**
     * Create a new knowledge base
     */
    public Knowledge.KnowledgeBase create(Knowledge.CreateKnowledgeBaseInput input) throws LinktorException {
        return http.post("/knowledge-bases", input, Knowledge.KnowledgeBase.class);
    }

    /**
     * Update a knowledge base
     */
    public Knowledge.KnowledgeBase update(String kbId, Knowledge.UpdateKnowledgeBaseInput input) throws LinktorException {
        return http.patch("/knowledge-bases/" + kbId, input, Knowledge.KnowledgeBase.class);
    }

    /**
     * Delete a knowledge base
     */
    public void delete(String kbId) throws LinktorException {
        http.delete("/knowledge-bases/" + kbId);
    }

    /**
     * List documents in a knowledge base
     */
    public Common.PaginatedResponse<Knowledge.Document> listDocuments(String kbId, Common.PaginationParams params) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (params != null) {
            if (params.getLimit() != null) queryParams.put("limit", params.getLimit().toString());
            if (params.getPage() != null) queryParams.put("page", params.getPage().toString());
        }

        Type responseType = new TypeToken<Common.PaginatedResponse<Knowledge.Document>>(){}.getType();
        return http.get("/knowledge-bases/" + kbId + "/documents", queryParams, responseType);
    }

    /**
     * List documents in a knowledge base (no pagination)
     */
    public Common.PaginatedResponse<Knowledge.Document> listDocuments(String kbId) throws LinktorException {
        return listDocuments(kbId, null);
    }

    /**
     * Add a document to a knowledge base
     */
    public Knowledge.Document addDocument(String kbId, Knowledge.AddDocumentInput input) throws LinktorException {
        return http.post("/knowledge-bases/" + kbId + "/documents", input, Knowledge.Document.class);
    }

    /**
     * Add a document by content
     */
    public Knowledge.Document addDocument(String kbId, String name, String content) throws LinktorException {
        Knowledge.AddDocumentInput input = Knowledge.AddDocumentInput.builder()
                .name(name)
                .content(content)
                .build();
        return addDocument(kbId, input);
    }

    /**
     * Get a document
     */
    public Knowledge.Document getDocument(String kbId, String documentId) throws LinktorException {
        return http.get("/knowledge-bases/" + kbId + "/documents/" + documentId, Knowledge.Document.class);
    }

    /**
     * Delete a document
     */
    public void deleteDocument(String kbId, String documentId) throws LinktorException {
        http.delete("/knowledge-bases/" + kbId + "/documents/" + documentId);
    }

    /**
     * Query a knowledge base
     */
    public Knowledge.QueryResult query(String kbId, String queryText, int topK) throws LinktorException {
        Knowledge.QueryKnowledgeBaseInput input = Knowledge.QueryKnowledgeBaseInput.builder()
                .query(queryText)
                .topK(topK)
                .build();
        return query(kbId, input);
    }

    /**
     * Query a knowledge base with full options
     */
    public Knowledge.QueryResult query(String kbId, Knowledge.QueryKnowledgeBaseInput input) throws LinktorException {
        return http.post("/knowledge-bases/" + kbId + "/query", input, Knowledge.QueryResult.class);
    }

    /**
     * Query a knowledge base (simple)
     */
    public Knowledge.QueryResult query(String kbId, String queryText) throws LinktorException {
        return query(kbId, queryText, 5);
    }
}
