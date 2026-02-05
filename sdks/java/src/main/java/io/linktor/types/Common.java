package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class Common {

    public static class PaginationParams {
        private Integer page;
        private Integer limit;
        private String cursor;
        private String sortBy;
        private String sortOrder;

        public Integer getPage() { return page; }
        public void setPage(Integer page) { this.page = page; }

        public Integer getLimit() { return limit; }
        public void setLimit(Integer limit) { this.limit = limit; }

        public String getCursor() { return cursor; }
        public void setCursor(String cursor) { this.cursor = cursor; }

        public String getSortBy() { return sortBy; }
        public void setSortBy(String sortBy) { this.sortBy = sortBy; }

        public String getSortOrder() { return sortOrder; }
        public void setSortOrder(String sortOrder) { this.sortOrder = sortOrder; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final PaginationParams params = new PaginationParams();

            public Builder page(Integer page) { params.page = page; return this; }
            public Builder limit(Integer limit) { params.limit = limit; return this; }
            public Builder cursor(String cursor) { params.cursor = cursor; return this; }
            public Builder sortBy(String sortBy) { params.sortBy = sortBy; return this; }
            public Builder sortOrder(String sortOrder) { params.sortOrder = sortOrder; return this; }
            public PaginationParams build() { return params; }
        }
    }

    public static class PaginatedResponse<T> {
        private List<T> data;
        private PaginationMeta pagination;

        public List<T> getData() { return data; }
        public void setData(List<T> data) { this.data = data; }

        public PaginationMeta getPagination() { return pagination; }
        public void setPagination(PaginationMeta pagination) { this.pagination = pagination; }
    }

    public static class PaginationMeta {
        private int total;
        private int page;
        private int limit;
        private int totalPages;
        private boolean hasMore;
        private String nextCursor;
        private String prevCursor;

        public int getTotal() { return total; }
        public void setTotal(int total) { this.total = total; }

        public int getPage() { return page; }
        public void setPage(int page) { this.page = page; }

        public int getLimit() { return limit; }
        public void setLimit(int limit) { this.limit = limit; }

        public int getTotalPages() { return totalPages; }
        public void setTotalPages(int totalPages) { this.totalPages = totalPages; }

        public boolean isHasMore() { return hasMore; }
        public void setHasMore(boolean hasMore) { this.hasMore = hasMore; }

        public String getNextCursor() { return nextCursor; }
        public void setNextCursor(String nextCursor) { this.nextCursor = nextCursor; }

        public String getPrevCursor() { return prevCursor; }
        public void setPrevCursor(String prevCursor) { this.prevCursor = prevCursor; }
    }

    public static class Timestamps {
        @SerializedName("createdAt")
        private Instant createdAt;

        @SerializedName("updatedAt")
        private Instant updatedAt;

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class ApiResponse<T> {
        private boolean success;
        private T data;
        private ApiError error;

        public boolean isSuccess() { return success; }
        public void setSuccess(boolean success) { this.success = success; }

        public T getData() { return data; }
        public void setData(T data) { this.data = data; }

        public ApiError getError() { return error; }
        public void setError(ApiError error) { this.error = error; }
    }

    public static class ApiError {
        private String code;
        private String message;
        private Map<String, Object> details;

        public String getCode() { return code; }
        public void setCode(String code) { this.code = code; }

        public String getMessage() { return message; }
        public void setMessage(String message) { this.message = message; }

        public Map<String, Object> getDetails() { return details; }
        public void setDetails(Map<String, Object> details) { this.details = details; }
    }
}
