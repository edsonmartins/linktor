package io.linktor.resources;

import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.util.HashMap;
import java.util.Map;

public class AnalyticsResource {
    private final HttpClient http;

    public AnalyticsResource(HttpClient http) {
        this.http = http;
    }

    /**
     * Get dashboard metrics
     */
    public DashboardMetrics getDashboard() throws LinktorException {
        return http.get("/analytics/dashboard", DashboardMetrics.class);
    }

    /**
     * Get dashboard metrics for a date range
     */
    public DashboardMetrics getDashboard(String startDate, String endDate) throws LinktorException {
        Map<String, String> params = new HashMap<>();
        if (startDate != null) params.put("startDate", startDate);
        if (endDate != null) params.put("endDate", endDate);
        return http.get("/analytics/dashboard", params, DashboardMetrics.class);
    }

    /**
     * Get real-time metrics
     */
    public RealtimeMetrics getRealtime() throws LinktorException {
        return http.get("/analytics/realtime", RealtimeMetrics.class);
    }

    /**
     * Get conversation statistics
     */
    public ConversationStats getConversationStats(String startDate, String endDate) throws LinktorException {
        Map<String, String> params = new HashMap<>();
        if (startDate != null) params.put("startDate", startDate);
        if (endDate != null) params.put("endDate", endDate);
        return http.get("/analytics/conversations", params, ConversationStats.class);
    }

    /**
     * Get agent performance metrics
     */
    public AgentPerformance getAgentPerformance(String startDate, String endDate) throws LinktorException {
        Map<String, String> params = new HashMap<>();
        if (startDate != null) params.put("startDate", startDate);
        if (endDate != null) params.put("endDate", endDate);
        return http.get("/analytics/agents", params, AgentPerformance.class);
    }

    /**
     * Get channel statistics
     */
    public ChannelStats getChannelStats(String startDate, String endDate) throws LinktorException {
        Map<String, String> params = new HashMap<>();
        if (startDate != null) params.put("startDate", startDate);
        if (endDate != null) params.put("endDate", endDate);
        return http.get("/analytics/channels", params, ChannelStats.class);
    }

    // Response classes
    public static class DashboardMetrics {
        private int totalConversations;
        private int openConversations;
        private int resolvedConversations;
        private int totalMessages;
        private int totalContacts;
        private double avgResponseTime;
        private double avgResolutionTime;
        private double satisfactionScore;
        private Map<String, Integer> conversationsByChannel;
        private Map<String, Integer> conversationsByStatus;

        public int getTotalConversations() { return totalConversations; }
        public void setTotalConversations(int totalConversations) { this.totalConversations = totalConversations; }

        public int getOpenConversations() { return openConversations; }
        public void setOpenConversations(int openConversations) { this.openConversations = openConversations; }

        public int getResolvedConversations() { return resolvedConversations; }
        public void setResolvedConversations(int resolvedConversations) { this.resolvedConversations = resolvedConversations; }

        public int getTotalMessages() { return totalMessages; }
        public void setTotalMessages(int totalMessages) { this.totalMessages = totalMessages; }

        public int getTotalContacts() { return totalContacts; }
        public void setTotalContacts(int totalContacts) { this.totalContacts = totalContacts; }

        public double getAvgResponseTime() { return avgResponseTime; }
        public void setAvgResponseTime(double avgResponseTime) { this.avgResponseTime = avgResponseTime; }

        public double getAvgResolutionTime() { return avgResolutionTime; }
        public void setAvgResolutionTime(double avgResolutionTime) { this.avgResolutionTime = avgResolutionTime; }

        public double getSatisfactionScore() { return satisfactionScore; }
        public void setSatisfactionScore(double satisfactionScore) { this.satisfactionScore = satisfactionScore; }

        public Map<String, Integer> getConversationsByChannel() { return conversationsByChannel; }
        public void setConversationsByChannel(Map<String, Integer> conversationsByChannel) { this.conversationsByChannel = conversationsByChannel; }

        public Map<String, Integer> getConversationsByStatus() { return conversationsByStatus; }
        public void setConversationsByStatus(Map<String, Integer> conversationsByStatus) { this.conversationsByStatus = conversationsByStatus; }
    }

    public static class RealtimeMetrics {
        private int activeConversations;
        private int onlineAgents;
        private int queuedConversations;
        private int messagesLastHour;
        private double avgWaitTime;

        public int getActiveConversations() { return activeConversations; }
        public void setActiveConversations(int activeConversations) { this.activeConversations = activeConversations; }

        public int getOnlineAgents() { return onlineAgents; }
        public void setOnlineAgents(int onlineAgents) { this.onlineAgents = onlineAgents; }

        public int getQueuedConversations() { return queuedConversations; }
        public void setQueuedConversations(int queuedConversations) { this.queuedConversations = queuedConversations; }

        public int getMessagesLastHour() { return messagesLastHour; }
        public void setMessagesLastHour(int messagesLastHour) { this.messagesLastHour = messagesLastHour; }

        public double getAvgWaitTime() { return avgWaitTime; }
        public void setAvgWaitTime(double avgWaitTime) { this.avgWaitTime = avgWaitTime; }
    }

    public static class ConversationStats {
        private int total;
        private int created;
        private int resolved;
        private double avgDuration;
        private Map<String, Integer> byChannel;
        private Map<String, Integer> byPriority;

        public int getTotal() { return total; }
        public void setTotal(int total) { this.total = total; }

        public int getCreated() { return created; }
        public void setCreated(int created) { this.created = created; }

        public int getResolved() { return resolved; }
        public void setResolved(int resolved) { this.resolved = resolved; }

        public double getAvgDuration() { return avgDuration; }
        public void setAvgDuration(double avgDuration) { this.avgDuration = avgDuration; }

        public Map<String, Integer> getByChannel() { return byChannel; }
        public void setByChannel(Map<String, Integer> byChannel) { this.byChannel = byChannel; }

        public Map<String, Integer> getByPriority() { return byPriority; }
        public void setByPriority(Map<String, Integer> byPriority) { this.byPriority = byPriority; }
    }

    public static class AgentPerformance {
        private int totalAgents;
        private int activeAgents;
        private double avgConversationsPerAgent;
        private double avgResponseTime;
        private double avgSatisfactionScore;

        public int getTotalAgents() { return totalAgents; }
        public void setTotalAgents(int totalAgents) { this.totalAgents = totalAgents; }

        public int getActiveAgents() { return activeAgents; }
        public void setActiveAgents(int activeAgents) { this.activeAgents = activeAgents; }

        public double getAvgConversationsPerAgent() { return avgConversationsPerAgent; }
        public void setAvgConversationsPerAgent(double avgConversationsPerAgent) { this.avgConversationsPerAgent = avgConversationsPerAgent; }

        public double getAvgResponseTime() { return avgResponseTime; }
        public void setAvgResponseTime(double avgResponseTime) { this.avgResponseTime = avgResponseTime; }

        public double getAvgSatisfactionScore() { return avgSatisfactionScore; }
        public void setAvgSatisfactionScore(double avgSatisfactionScore) { this.avgSatisfactionScore = avgSatisfactionScore; }
    }

    public static class ChannelStats {
        private Map<String, ChannelMetrics> channels;

        public Map<String, ChannelMetrics> getChannels() { return channels; }
        public void setChannels(Map<String, ChannelMetrics> channels) { this.channels = channels; }
    }

    public static class ChannelMetrics {
        private int conversations;
        private int messages;
        private double avgResponseTime;

        public int getConversations() { return conversations; }
        public void setConversations(int conversations) { this.conversations = conversations; }

        public int getMessages() { return messages; }
        public void setMessages(int messages) { this.messages = messages; }

        public double getAvgResponseTime() { return avgResponseTime; }
        public void setAvgResponseTime(double avgResponseTime) { this.avgResponseTime = avgResponseTime; }
    }
}
