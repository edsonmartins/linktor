package io.linktor.utils;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonDeserializer;
import com.google.gson.JsonSerializer;
import io.linktor.types.Common;
import okhttp3.*;

import java.io.IOException;
import java.lang.reflect.Type;
import java.time.Instant;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class HttpClient {
    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");

    private final OkHttpClient client;
    private final String baseUrl;
    private final Gson gson;

    private String apiKey;
    private String accessToken;
    private final int maxRetries;

    public HttpClient(String baseUrl, String apiKey, String accessToken, int timeoutSeconds, int maxRetries) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.apiKey = apiKey;
        this.accessToken = accessToken;
        this.maxRetries = maxRetries;

        this.client = new OkHttpClient.Builder()
                .connectTimeout(timeoutSeconds, TimeUnit.SECONDS)
                .readTimeout(timeoutSeconds, TimeUnit.SECONDS)
                .writeTimeout(timeoutSeconds, TimeUnit.SECONDS)
                .build();

        this.gson = new GsonBuilder()
                .registerTypeAdapter(Instant.class, (JsonSerializer<Instant>) (src, typeOfSrc, context) ->
                        context.serialize(src.toString()))
                .registerTypeAdapter(Instant.class, (JsonDeserializer<Instant>) (json, typeOfT, context) ->
                        Instant.parse(json.getAsString()))
                .create();
    }

    public void setApiKey(String apiKey) {
        this.apiKey = apiKey;
    }

    public void setAccessToken(String accessToken) {
        this.accessToken = accessToken;
    }

    public <T> T get(String path, Type responseType) throws LinktorException {
        return request("GET", path, null, responseType);
    }

    public <T> T get(String path, Map<String, String> queryParams, Type responseType) throws LinktorException {
        String url = buildUrlWithParams(path, queryParams);
        return request("GET", url, null, responseType);
    }

    public <T> T post(String path, Object body, Type responseType) throws LinktorException {
        return request("POST", path, body, responseType);
    }

    public <T> T put(String path, Object body, Type responseType) throws LinktorException {
        return request("PUT", path, body, responseType);
    }

    public <T> T patch(String path, Object body, Type responseType) throws LinktorException {
        return request("PATCH", path, body, responseType);
    }

    public void delete(String path) throws LinktorException {
        request("DELETE", path, null, Void.class);
    }

    private <T> T request(String method, String path, Object body, Type responseType) throws LinktorException {
        String url = path.startsWith("http") ? path : baseUrl + path;

        Request.Builder requestBuilder = new Request.Builder()
                .url(url)
                .header("Content-Type", "application/json")
                .header("Accept", "application/json");

        // Add authentication
        if (apiKey != null && !apiKey.isEmpty()) {
            requestBuilder.header("X-API-Key", apiKey);
        } else if (accessToken != null && !accessToken.isEmpty()) {
            requestBuilder.header("Authorization", "Bearer " + accessToken);
        }

        // Add body
        RequestBody requestBody = null;
        if (body != null) {
            requestBody = RequestBody.create(gson.toJson(body), JSON);
        }

        switch (method.toUpperCase()) {
            case "GET":
                requestBuilder.get();
                break;
            case "POST":
                requestBuilder.post(requestBody != null ? requestBody : RequestBody.create("", JSON));
                break;
            case "PUT":
                requestBuilder.put(requestBody != null ? requestBody : RequestBody.create("", JSON));
                break;
            case "PATCH":
                requestBuilder.patch(requestBody != null ? requestBody : RequestBody.create("", JSON));
                break;
            case "DELETE":
                requestBuilder.delete(requestBody);
                break;
        }

        Request request = requestBuilder.build();

        // Retry logic
        int attempts = 0;
        LinktorException lastException = null;

        while (attempts < maxRetries) {
            attempts++;

            try {
                Response response = client.newCall(request).execute();
                return handleResponse(response, responseType);
            } catch (IOException e) {
                lastException = new LinktorException.NetworkException("Network error: " + e.getMessage(), e);
            } catch (LinktorException.RateLimitException e) {
                lastException = e;
                if (attempts < maxRetries) {
                    try {
                        Thread.sleep(e.getRetryAfter() * 1000);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw e;
                    }
                }
            } catch (LinktorException.ServerException e) {
                lastException = e;
                if (attempts < maxRetries) {
                    try {
                        Thread.sleep((long) Math.pow(2, attempts) * 1000);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw e;
                    }
                }
            }
        }

        throw lastException != null ? lastException : new LinktorException("Request failed after " + maxRetries + " attempts");
    }

    @SuppressWarnings("unchecked")
    private <T> T handleResponse(Response response, Type responseType) throws LinktorException {
        String requestId = response.header("X-Request-ID");

        try {
            String bodyString = response.body() != null ? response.body().string() : "";

            if (!response.isSuccessful()) {
                handleError(response.code(), bodyString, requestId);
            }

            if (responseType == Void.class || bodyString.isEmpty()) {
                return null;
            }

            // Check if response is wrapped in ApiResponse
            if (bodyString.startsWith("{\"success\":")) {
                Common.ApiResponse<?> apiResponse = gson.fromJson(bodyString,
                        com.google.gson.reflect.TypeToken.getParameterized(Common.ApiResponse.class, responseType).getType());
                if (apiResponse.isSuccess()) {
                    return (T) apiResponse.getData();
                } else {
                    Common.ApiError error = apiResponse.getError();
                    throw new LinktorException(
                            error != null ? error.getMessage() : "Unknown error",
                            response.code(),
                            error != null ? error.getCode() : null,
                            requestId
                    );
                }
            }

            return gson.fromJson(bodyString, responseType);
        } catch (IOException e) {
            throw new LinktorException.NetworkException("Failed to read response body", e);
        }
    }

    private void handleError(int statusCode, String body, String requestId) throws LinktorException {
        String message = "Request failed";

        try {
            Common.ApiError error = gson.fromJson(body, Common.ApiError.class);
            if (error != null && error.getMessage() != null) {
                message = error.getMessage();
            }
        } catch (Exception e) {
            // Use default message
        }

        switch (statusCode) {
            case 400:
                throw new LinktorException.ValidationException(message, requestId);
            case 401:
                throw new LinktorException.AuthenticationException(message, requestId);
            case 403:
                throw new LinktorException.AuthorizationException(message, requestId);
            case 404:
                throw new LinktorException.NotFoundException(message, requestId);
            case 429:
                throw new LinktorException.RateLimitException(message, 60, requestId);
            case 500:
            case 502:
            case 503:
            case 504:
                throw new LinktorException.ServerException(message, requestId);
            default:
                throw new LinktorException(message, statusCode, null, requestId);
        }
    }

    private String buildUrlWithParams(String path, Map<String, String> params) {
        if (params == null || params.isEmpty()) {
            return path;
        }

        StringBuilder url = new StringBuilder(path);
        boolean first = !path.contains("?");

        for (Map.Entry<String, String> entry : params.entrySet()) {
            if (entry.getValue() != null) {
                url.append(first ? "?" : "&");
                url.append(entry.getKey()).append("=").append(entry.getValue());
                first = false;
            }
        }

        return url.toString();
    }

    public Gson getGson() {
        return gson;
    }
}
