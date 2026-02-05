/**
 * Channel types
 */

import type { ChannelType, PaginationParams, Timestamps } from './common';

export interface Channel extends Timestamps {
  id: string;
  tenantId: string;
  name: string;
  type: ChannelType;
  status: ChannelStatus;
  config: ChannelConfig;
  webhookUrl?: string;
  metadata?: Record<string, unknown>;
}

export type ChannelStatus = 'active' | 'inactive' | 'connecting' | 'error';

export type ChannelConfig =
  | WhatsAppConfig
  | TelegramConfig
  | FacebookConfig
  | InstagramConfig
  | WebchatConfig
  | SMSConfig
  | EmailConfig
  | RCSConfig;

export interface WhatsAppConfig {
  type: 'whatsapp';
  phoneNumberId: string;
  businessAccountId: string;
  accessToken: string;
  verifyToken: string;
  appSecret?: string;
}

export interface TelegramConfig {
  type: 'telegram';
  botToken: string;
  botUsername?: string;
  webhookSecret?: string;
}

export interface FacebookConfig {
  type: 'facebook';
  pageId: string;
  pageAccessToken: string;
  appSecret?: string;
  pageName?: string;
  pageProfilePicture?: string;
}

export interface InstagramConfig {
  type: 'instagram';
  instagramId: string;
  pageAccessToken: string;
  appSecret?: string;
  username?: string;
  profilePicture?: string;
}

export interface WebchatConfig {
  type: 'webchat';
  widgetId: string;
  allowedOrigins?: string[];
  theme?: WebchatTheme;
}

export interface WebchatTheme {
  primaryColor?: string;
  headerText?: string;
  welcomeMessage?: string;
  position?: 'left' | 'right';
}

export interface SMSConfig {
  type: 'sms';
  provider: 'twilio';
  accountSid: string;
  authToken: string;
  phoneNumber: string;
}

export interface EmailConfig {
  type: 'email';
  provider: 'smtp' | 'sendgrid' | 'mailgun' | 'ses' | 'postmark';
  fromEmail: string;
  fromName?: string;
  // Provider-specific config
  smtpHost?: string;
  smtpPort?: number;
  smtpUsername?: string;
  smtpPassword?: string;
  apiKey?: string;
  domain?: string;
  region?: string;
}

export interface RCSConfig {
  type: 'rcs';
  provider: 'zenvia' | 'infobip' | 'pontaltech';
  agentId: string;
  apiKey: string;
  brandName?: string;
}

// Request types
export interface CreateChannelInput {
  name: string;
  type: ChannelType;
  config: ChannelConfig;
  metadata?: Record<string, unknown>;
}

export interface UpdateChannelInput {
  name?: string;
  config?: Partial<ChannelConfig>;
  metadata?: Record<string, unknown>;
}

export interface ListChannelsParams extends PaginationParams {
  type?: ChannelType;
  status?: ChannelStatus;
  search?: string;
}

export interface ChannelCapabilities {
  supportedContentTypes: string[];
  supportsMedia: boolean;
  supportsButtons: boolean;
  supportsLists: boolean;
  supportsTemplates: boolean;
  supportsLocation: boolean;
  maxMessageLength: number;
  maxMediaSize: number;
}
