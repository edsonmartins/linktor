package io.linktor.utils;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonDeserializer;
import io.linktor.types.Webhook;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.InvalidKeyException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.Map;

public class WebhookVerifier {
    private static final String HMAC_SHA256 = "HmacSHA256";
    private static final Gson gson = new GsonBuilder()
            .registerTypeAdapter(Instant.class, (JsonDeserializer<Instant>) (json, typeOfT, context) ->
                    Instant.parse(json.getAsString()))
            .create();

    /**
     * Compute HMAC-SHA256 signature for the given payload
     */
    public static String computeSignature(byte[] payload, String secret) {
        try {
            Mac mac = Mac.getInstance(HMAC_SHA256);
            SecretKeySpec secretKey = new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), HMAC_SHA256);
            mac.init(secretKey);
            byte[] hash = mac.doFinal(payload);
            return bytesToHex(hash);
        } catch (NoSuchAlgorithmException | InvalidKeyException e) {
            throw new RuntimeException("Failed to compute signature", e);
        }
    }

    /**
     * Compute HMAC-SHA256 signature for the given payload string
     */
    public static String computeSignature(String payload, String secret) {
        return computeSignature(payload.getBytes(StandardCharsets.UTF_8), secret);
    }

    /**
     * Verify webhook signature only (no timestamp validation)
     */
    public static boolean verifySignature(byte[] payload, String signature, String secret) {
        if (signature == null || signature.isEmpty() || secret == null || secret.isEmpty()) {
            return false;
        }

        String expected = computeSignature(payload, secret);
        return MessageDigest.isEqual(
                signature.getBytes(StandardCharsets.UTF_8),
                expected.getBytes(StandardCharsets.UTF_8)
        );
    }

    /**
     * Verify webhook signature only (no timestamp validation)
     */
    public static boolean verifySignature(String payload, String signature, String secret) {
        return verifySignature(payload.getBytes(StandardCharsets.UTF_8), signature, secret);
    }

    /**
     * Verify webhook with signature and timestamp validation
     */
    public static boolean verify(byte[] payload, Map<String, String> headers, String secret, int toleranceSeconds) {
        String signature = headers.get(Webhook.SIGNATURE_HEADER);
        if (signature == null) {
            signature = headers.get(Webhook.SIGNATURE_HEADER.toLowerCase());
        }

        if (signature == null || signature.isEmpty()) {
            return false;
        }

        // Verify timestamp if present
        String timestampStr = headers.get(Webhook.TIMESTAMP_HEADER);
        if (timestampStr == null) {
            timestampStr = headers.get(Webhook.TIMESTAMP_HEADER.toLowerCase());
        }

        if (timestampStr != null && !timestampStr.isEmpty()) {
            try {
                long timestamp = Long.parseLong(timestampStr);
                long now = Instant.now().getEpochSecond();
                if (Math.abs(now - timestamp) > toleranceSeconds) {
                    return false;
                }
            } catch (NumberFormatException e) {
                return false;
            }
        }

        return verifySignature(payload, signature, secret);
    }

    /**
     * Verify webhook with signature and default timestamp tolerance (5 minutes)
     */
    public static boolean verify(byte[] payload, Map<String, String> headers, String secret) {
        return verify(payload, headers, secret, Webhook.DEFAULT_TOLERANCE_SECONDS);
    }

    /**
     * Construct and verify a webhook event
     */
    public static Webhook.WebhookEvent constructEvent(byte[] payload, Map<String, String> headers, String secret, int toleranceSeconds)
            throws LinktorException.WebhookVerificationException {
        if (!verify(payload, headers, secret, toleranceSeconds == 0 ? Webhook.DEFAULT_TOLERANCE_SECONDS : toleranceSeconds)) {
            throw new LinktorException.WebhookVerificationException("Webhook signature verification failed");
        }

        try {
            String json = new String(payload, StandardCharsets.UTF_8);
            Webhook.WebhookEvent event = gson.fromJson(json, Webhook.WebhookEvent.class);

            if (event.getId() == null || event.getType() == null) {
                throw new LinktorException.WebhookVerificationException("Invalid webhook event structure");
            }

            return event;
        } catch (Exception e) {
            if (e instanceof LinktorException.WebhookVerificationException) {
                throw (LinktorException.WebhookVerificationException) e;
            }
            throw new LinktorException.WebhookVerificationException("Failed to parse webhook event: " + e.getMessage());
        }
    }

    /**
     * Construct and verify a webhook event with default tolerance
     */
    public static Webhook.WebhookEvent constructEvent(byte[] payload, Map<String, String> headers, String secret)
            throws LinktorException.WebhookVerificationException {
        return constructEvent(payload, headers, secret, Webhook.DEFAULT_TOLERANCE_SECONDS);
    }

    /**
     * Construct and verify a webhook event from string payload
     */
    public static Webhook.WebhookEvent constructEvent(String payload, Map<String, String> headers, String secret)
            throws LinktorException.WebhookVerificationException {
        return constructEvent(payload.getBytes(StandardCharsets.UTF_8), headers, secret, Webhook.DEFAULT_TOLERANCE_SECONDS);
    }

    private static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder();
        for (byte b : bytes) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }
}
