"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  MessageSquare,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Loader2,
  Settings,
  TestTube,
  Link2,
  Shield,
} from "lucide-react";
import { useToast } from "@/hooks/use-toast";

// Provider types
type RCSProvider = "zenvia" | "infobip" | "pontaltech" | "google";

interface RCSConfigProps {
  channelId?: string;
  initialConfig?: RCSChannelConfig;
  onSave?: (config: RCSChannelConfig) => void;
}

interface RCSChannelConfig {
  provider: RCSProvider;
  agentId: string;
  apiKey: string;
  apiSecret?: string;
  baseUrl?: string;
  senderId?: string;
  webhookUrl?: string;
  webhookSecret?: string;
}

// Provider-specific schemas
const zenviaSchema = z.object({
  provider: z.literal("zenvia"),
  agentId: z.string().min(1, "Agent ID is required"),
  apiKey: z.string().min(1, "API Token is required"),
  senderId: z.string().optional(),
});

const infobipSchema = z.object({
  provider: z.literal("infobip"),
  agentId: z.string().min(1, "Agent ID is required"),
  apiKey: z.string().min(1, "API Key is required"),
  baseUrl: z.string().url("Invalid base URL").min(1, "Base URL is required"),
});

const pontaltechSchema = z.object({
  provider: z.literal("pontaltech"),
  agentId: z.string().min(1, "Agent ID is required"),
  apiKey: z.string().min(1, "API Key is required"),
  apiSecret: z.string().min(1, "API Secret is required"),
});

const googleSchema = z.object({
  provider: z.literal("google"),
  agentId: z.string().min(1, "Agent ID is required"),
  apiKey: z.string().min(1, "Service Account JSON is required"),
});

// Combined schema
const formSchema = z.discriminatedUnion("provider", [
  zenviaSchema,
  infobipSchema,
  pontaltechSchema,
  googleSchema,
]);

type FormData = z.infer<typeof formSchema>;

const providerInfo: Record<RCSProvider, { name: string; description: string; docsUrl: string }> = {
  zenvia: {
    name: "Zenvia",
    description: "Brazilian CPaaS provider with RCS Business Messaging support",
    docsUrl: "https://zenvia.github.io/zenvia-openapi-spec/",
  },
  infobip: {
    name: "Infobip",
    description: "Global cloud communications platform with RCS capabilities",
    docsUrl: "https://www.infobip.com/docs/rcs",
  },
  pontaltech: {
    name: "Pontaltech",
    description: "Brazilian messaging provider specializing in RCS",
    docsUrl: "https://www.pontaltech.com.br/",
  },
  google: {
    name: "Google RCS Business Messaging",
    description: "Direct integration with Google's RCS Business Messaging API",
    docsUrl: "https://developers.google.com/business-communications/rcs-business-messaging",
  },
};

export function RCSConfig({ channelId, initialConfig, onSave }: RCSConfigProps) {
  const [selectedProvider, setSelectedProvider] = useState<RCSProvider>(
    initialConfig?.provider || "zenvia"
  );
  const [connectionStatus, setConnectionStatus] = useState<"idle" | "connecting" | "connected" | "error">("idle");
  const [testStatus, setTestStatus] = useState<"idle" | "testing" | "success" | "error">("idle");
  const [webhookUrl, setWebhookUrl] = useState<string>("");
  const { toast } = useToast();

  const form = useForm<FormData>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      provider: selectedProvider,
      agentId: initialConfig?.agentId || "",
      apiKey: initialConfig?.apiKey || "",
      apiSecret: initialConfig?.apiSecret || "",
      baseUrl: initialConfig?.baseUrl || "",
      senderId: initialConfig?.senderId || "",
    } as FormData,
  });

  const handleProviderChange = (provider: RCSProvider) => {
    setSelectedProvider(provider);
    form.setValue("provider", provider);
    setConnectionStatus("idle");
    setTestStatus("idle");
  };

  const onSubmit = async (data: FormData) => {
    setConnectionStatus("connecting");

    try {
      const response = await fetch(`/api/channels/rcs${channelId ? `/${channelId}` : ""}`, {
        method: channelId ? "PUT" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        throw new Error("Failed to save configuration");
      }

      const result = await response.json();
      setWebhookUrl(result.webhookUrl || "");
      setConnectionStatus("connected");

      toast({
        title: "Configuration saved",
        description: "RCS channel has been configured successfully.",
      });

      if (onSave) {
        onSave(data as RCSChannelConfig);
      }
    } catch (error) {
      setConnectionStatus("error");
      toast({
        variant: "error",
        title: "Configuration failed",
        description: error instanceof Error ? error.message : "Failed to save configuration",
      });
    }
  };

  const handleTestConnection = async () => {
    setTestStatus("testing");

    try {
      const values = form.getValues();
      const response = await fetch("/api/channels/rcs/test", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });

      if (!response.ok) {
        throw new Error("Connection test failed");
      }

      setTestStatus("success");
      toast({
        title: "Connection successful",
        description: "RCS provider connection verified.",
      });
    } catch (error) {
      setTestStatus("error");
      toast({
        variant: "error",
        title: "Test failed",
        description: error instanceof Error ? error.message : "Connection test failed",
      });
    }
  };

  const renderProviderFields = () => {
    switch (selectedProvider) {
      case "zenvia":
        return (
          <>
            <FormField
              control={form.control}
              name="agentId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Agent ID</FormLabel>
                  <FormControl>
                    <Input placeholder="your-agent-id" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Zenvia RCS Agent ID from the console
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="apiKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>API Token</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="Enter your API token" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Zenvia API token for authentication
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="senderId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Sender ID (Optional)</FormLabel>
                  <FormControl>
                    <Input placeholder="YourBrand" {...field} />
                  </FormControl>
                  <FormDescription>
                    Custom sender ID for outgoing messages
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </>
        );

      case "infobip":
        return (
          <>
            <FormField
              control={form.control}
              name="agentId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Agent ID</FormLabel>
                  <FormControl>
                    <Input placeholder="your-agent-id" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Infobip RCS Agent ID
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="apiKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>API Key</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="Enter your API key" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Infobip API key from the dashboard
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="baseUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Base URL</FormLabel>
                  <FormControl>
                    <Input placeholder="https://xxxxx.api.infobip.com" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Infobip API base URL (region-specific)
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </>
        );

      case "pontaltech":
        return (
          <>
            <FormField
              control={form.control}
              name="agentId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Agent ID</FormLabel>
                  <FormControl>
                    <Input placeholder="your-agent-id" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Pontaltech RCS Agent ID
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="apiKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>API Key</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="Enter your API key" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Pontaltech API key
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="apiSecret"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>API Secret</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="Enter your API secret" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Pontaltech API secret for signing requests
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </>
        );

      case "google":
        return (
          <>
            <FormField
              control={form.control}
              name="agentId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Agent ID</FormLabel>
                  <FormControl>
                    <Input placeholder="brands/BRAND_ID/agents/AGENT_ID" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Google RCS Business Messaging Agent ID
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="apiKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Service Account JSON</FormLabel>
                  <FormControl>
                    <textarea
                      className="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                      placeholder='{"type": "service_account", ...}'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Paste your Google Cloud service account JSON credentials
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </>
        );

      default:
        return null;
    }
  };

  return (
    <div className="space-y-6">
      {/* Provider Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5" />
            RCS Provider
          </CardTitle>
          <CardDescription>
            Select your RCS Business Messaging provider
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {(Object.keys(providerInfo) as RCSProvider[]).map((provider) => (
              <Card
                key={provider}
                className={`cursor-pointer transition-all ${
                  selectedProvider === provider
                    ? "border-primary ring-2 ring-primary"
                    : "hover:border-muted-foreground"
                }`}
                onClick={() => handleProviderChange(provider)}
              >
                <CardHeader className="p-4">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base">{providerInfo[provider].name}</CardTitle>
                    {selectedProvider === provider && (
                      <CheckCircle2 className="h-5 w-5 text-primary" />
                    )}
                  </div>
                  <CardDescription className="text-xs">
                    {providerInfo[provider].description}
                  </CardDescription>
                </CardHeader>
              </Card>
            ))}
          </div>

          <div className="mt-4 p-3 bg-muted rounded-lg">
            <p className="text-sm text-muted-foreground">
              <a
                href={providerInfo[selectedProvider].docsUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline inline-flex items-center gap-1"
              >
                <Link2 className="h-3 w-3" />
                View {providerInfo[selectedProvider].name} Documentation
              </a>
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Configuration Form */}
      <Tabs defaultValue="credentials">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="credentials" className="flex items-center gap-2">
            <Settings className="h-4 w-4" />
            Credentials
          </TabsTrigger>
          <TabsTrigger value="webhook" className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            Webhook
          </TabsTrigger>
        </TabsList>

        <TabsContent value="credentials">
          <Card>
            <CardHeader>
              <CardTitle>Provider Credentials</CardTitle>
              <CardDescription>
                Enter your {providerInfo[selectedProvider].name} credentials
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                  {renderProviderFields()}

                  <div className="flex gap-2 pt-4">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleTestConnection}
                      disabled={testStatus === "testing"}
                    >
                      {testStatus === "testing" ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Testing...
                        </>
                      ) : (
                        <>
                          <TestTube className="mr-2 h-4 w-4" />
                          Test Connection
                        </>
                      )}
                    </Button>
                    <Button type="submit" disabled={connectionStatus === "connecting"}>
                      {connectionStatus === "connecting" ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Saving...
                        </>
                      ) : (
                        "Save Configuration"
                      )}
                    </Button>
                  </div>

                  {testStatus === "success" && (
                    <Alert className="mt-4">
                      <CheckCircle2 className="h-4 w-4" />
                      <AlertTitle>Connection Verified</AlertTitle>
                      <AlertDescription>
                        Successfully connected to {providerInfo[selectedProvider].name}.
                      </AlertDescription>
                    </Alert>
                  )}

                  {testStatus === "error" && (
                    <Alert variant="destructive" className="mt-4">
                      <XCircle className="h-4 w-4" />
                      <AlertTitle>Connection Failed</AlertTitle>
                      <AlertDescription>
                        Failed to connect. Please verify your credentials.
                      </AlertDescription>
                    </Alert>
                  )}
                </form>
              </Form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="webhook">
          <Card>
            <CardHeader>
              <CardTitle>Webhook Configuration</CardTitle>
              <CardDescription>
                Configure your webhook URL in the {providerInfo[selectedProvider].name} dashboard
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {connectionStatus === "connected" && webhookUrl ? (
                <>
                  <div className="space-y-2">
                    <Label>Webhook URL</Label>
                    <div className="flex gap-2">
                      <Input value={webhookUrl} readOnly className="font-mono text-sm" />
                      <Button
                        variant="outline"
                        onClick={() => {
                          navigator.clipboard.writeText(webhookUrl);
                          toast({
                            title: "Copied",
                            description: "Webhook URL copied to clipboard",
                          });
                        }}
                      >
                        Copy
                      </Button>
                    </div>
                  </div>

                  <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertTitle>Setup Instructions</AlertTitle>
                    <AlertDescription className="space-y-2">
                      <p>Configure this webhook URL in your {providerInfo[selectedProvider].name} dashboard:</p>
                      <ol className="list-decimal list-inside space-y-1 text-sm">
                        {selectedProvider === "zenvia" && (
                          <>
                            <li>Go to your Zenvia console</li>
                            <li>Navigate to your RCS agent settings</li>
                            <li>Set the webhook URL above as the callback URL</li>
                            <li>Enable message and status callbacks</li>
                          </>
                        )}
                        {selectedProvider === "infobip" && (
                          <>
                            <li>Go to your Infobip dashboard</li>
                            <li>Navigate to Channels {"->"} RCS {"->"} Webhooks</li>
                            <li>Add the webhook URL for incoming messages</li>
                            <li>Add the webhook URL for delivery reports</li>
                          </>
                        )}
                        {selectedProvider === "pontaltech" && (
                          <>
                            <li>Contact Pontaltech support</li>
                            <li>Provide the webhook URL for configuration</li>
                            <li>Request webhook secret for validation</li>
                          </>
                        )}
                        {selectedProvider === "google" && (
                          <>
                            <li>Go to Google Business Communications console</li>
                            <li>Navigate to your RCS agent settings</li>
                            <li>Set the webhook URL in Agent Information</li>
                            <li>Verify webhook ownership</li>
                          </>
                        )}
                      </ol>
                    </AlertDescription>
                  </Alert>
                </>
              ) : (
                <Alert>
                  <AlertCircle className="h-4 w-4" />
                  <AlertTitle>Save Configuration First</AlertTitle>
                  <AlertDescription>
                    Save your credentials to get the webhook URL for your channel.
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Connection Status */}
      <Card>
        <CardHeader>
          <CardTitle>Connection Status</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            {connectionStatus === "idle" && (
              <>
                <div className="h-3 w-3 rounded-full bg-muted" />
                <span className="text-muted-foreground">Not configured</span>
              </>
            )}
            {connectionStatus === "connecting" && (
              <>
                <Loader2 className="h-4 w-4 animate-spin text-primary" />
                <span>Connecting...</span>
              </>
            )}
            {connectionStatus === "connected" && (
              <>
                <div className="h-3 w-3 rounded-full bg-green-500" />
                <span className="text-green-600">Connected</span>
                <Badge variant="outline" className="ml-2">
                  {providerInfo[selectedProvider].name}
                </Badge>
              </>
            )}
            {connectionStatus === "error" && (
              <>
                <div className="h-3 w-3 rounded-full bg-destructive" />
                <span className="text-destructive">Connection failed</span>
              </>
            )}
          </div>
        </CardContent>
      </Card>

      {/* RCS Features Info */}
      <Card>
        <CardHeader>
          <CardTitle>RCS Features</CardTitle>
          <CardDescription>
            Rich Communication Services capabilities
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">💬</div>
              <div className="text-xs font-medium">Rich Text</div>
            </div>
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">🖼️</div>
              <div className="text-xs font-medium">Rich Cards</div>
            </div>
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">🎠</div>
              <div className="text-xs font-medium">Carousels</div>
            </div>
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">✅</div>
              <div className="text-xs font-medium">Read Receipts</div>
            </div>
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">📍</div>
              <div className="text-xs font-medium">Location</div>
            </div>
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">🔘</div>
              <div className="text-xs font-medium">Quick Replies</div>
            </div>
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">📎</div>
              <div className="text-xs font-medium">Media</div>
            </div>
            <div className="text-center p-3 bg-muted rounded-lg">
              <div className="text-2xl mb-1">⌨️</div>
              <div className="text-xs font-medium">Typing Indicator</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
