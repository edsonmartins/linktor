package plugin

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Note: In production, you would generate these from .proto files
// For now, we'll create manual implementations that can be replaced later

// GRPCServer is the gRPC server implementation for the plugin
type GRPCServer struct {
	UnimplementedChannelAdapterServer
	Impl ChannelAdapter
}

// Initialize implements the gRPC Initialize method
func (s *GRPCServer) Initialize(ctx context.Context, req *InitializeRequest) (*emptypb.Empty, error) {
	err := s.Impl.Initialize(req.Config)
	return &emptypb.Empty{}, err
}

// Connect implements the gRPC Connect method
func (s *GRPCServer) Connect(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.Impl.Connect(ctx)
	return &emptypb.Empty{}, err
}

// Disconnect implements the gRPC Disconnect method
func (s *GRPCServer) Disconnect(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.Impl.Disconnect(ctx)
	return &emptypb.Empty{}, err
}

// IsConnected implements the gRPC IsConnected method
func (s *GRPCServer) IsConnected(ctx context.Context, _ *emptypb.Empty) (*IsConnectedResponse, error) {
	connected := s.Impl.IsConnected()
	return &IsConnectedResponse{Connected: connected}, nil
}

// SendMessage implements the gRPC SendMessage method
func (s *GRPCServer) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendResultResponse, error) {
	msg := &OutboundMessage{
		ID:             req.Id,
		ConversationID: req.ConversationId,
		RecipientID:    req.RecipientId,
		ContentType:    ContentType(req.ContentType),
		Content:        req.Content,
		Metadata:       req.Metadata,
	}

	for _, att := range req.Attachments {
		msg.Attachments = append(msg.Attachments, &Attachment{
			Type:         att.Type,
			URL:          att.Url,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
			SizeBytes:    att.SizeBytes,
			ThumbnailURL: att.ThumbnailUrl,
			Metadata:     att.Metadata,
		})
	}

	result, err := s.Impl.SendMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &SendResultResponse{
		Success:    result.Success,
		ExternalId: result.ExternalID,
		Status:     string(result.Status),
		Error:      result.Error,
		Timestamp:  timestamppb.New(result.Timestamp),
	}, nil
}

// GetChannelInfo implements the gRPC GetChannelInfo method
func (s *GRPCServer) GetChannelInfo(ctx context.Context, _ *emptypb.Empty) (*ChannelInfoResponse, error) {
	info := s.Impl.GetChannelInfo()

	caps := &CapabilitiesResponse{}
	if info.Capabilities != nil {
		for _, ct := range info.Capabilities.SupportedContentTypes {
			caps.SupportedContentTypes = append(caps.SupportedContentTypes, string(ct))
		}
		caps.SupportsMedia = info.Capabilities.SupportsMedia
		caps.SupportsLocation = info.Capabilities.SupportsLocation
		caps.SupportsTemplates = info.Capabilities.SupportsTemplates
		caps.SupportsInteractive = info.Capabilities.SupportsInteractive
		caps.SupportsReadReceipts = info.Capabilities.SupportsReadReceipts
		caps.MaxMessageLength = int32(info.Capabilities.MaxMessageLength)
		caps.MaxMediaSize = info.Capabilities.MaxMediaSize
	}

	return &ChannelInfoResponse{
		Type:         string(info.Type),
		Name:         info.Name,
		Description:  info.Description,
		Version:      info.Version,
		Author:       info.Author,
		Capabilities: caps,
	}, nil
}

// GRPCClient is the gRPC client implementation for the plugin
type GRPCClient struct {
	client ChannelAdapterClient
}

// Initialize implements the ChannelAdapter interface
func (c *GRPCClient) Initialize(config map[string]string) error {
	_, err := c.client.Initialize(context.Background(), &InitializeRequest{Config: config})
	return err
}

// Connect implements the ChannelAdapter interface
func (c *GRPCClient) Connect(ctx context.Context) error {
	_, err := c.client.Connect(ctx, &emptypb.Empty{})
	return err
}

// Disconnect implements the ChannelAdapter interface
func (c *GRPCClient) Disconnect(ctx context.Context) error {
	_, err := c.client.Disconnect(ctx, &emptypb.Empty{})
	return err
}

// IsConnected implements the ChannelAdapter interface
func (c *GRPCClient) IsConnected() bool {
	resp, err := c.client.IsConnected(context.Background(), &emptypb.Empty{})
	if err != nil {
		return false
	}
	return resp.Connected
}

// GetConnectionStatus implements the ChannelAdapter interface
func (c *GRPCClient) GetConnectionStatus() *ConnectionStatus {
	return &ConnectionStatus{Connected: c.IsConnected()}
}

// SendMessage implements the ChannelAdapter interface
func (c *GRPCClient) SendMessage(ctx context.Context, msg *OutboundMessage) (*SendResult, error) {
	attachments := make([]*AttachmentProto, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, &AttachmentProto{
			Type:         att.Type,
			Url:          att.URL,
			Filename:     att.Filename,
			MimeType:     att.MimeType,
			SizeBytes:    att.SizeBytes,
			ThumbnailUrl: att.ThumbnailURL,
			Metadata:     att.Metadata,
		})
	}

	resp, err := c.client.SendMessage(ctx, &SendMessageRequest{
		Id:             msg.ID,
		ConversationId: msg.ConversationID,
		RecipientId:    msg.RecipientID,
		ContentType:    string(msg.ContentType),
		Content:        msg.Content,
		Metadata:       msg.Metadata,
		Attachments:    attachments,
	})
	if err != nil {
		return nil, err
	}

	return &SendResult{
		Success:    resp.Success,
		ExternalID: resp.ExternalId,
		Status:     MessageStatus(resp.Status),
		Error:      resp.Error,
		Timestamp:  resp.Timestamp.AsTime(),
	}, nil
}

// SendTypingIndicator implements the ChannelAdapter interface
func (c *GRPCClient) SendTypingIndicator(ctx context.Context, indicator *TypingIndicator) error {
	return nil // Not implemented in basic gRPC version
}

// SendReadReceipt implements the ChannelAdapter interface
func (c *GRPCClient) SendReadReceipt(ctx context.Context, receipt *ReadReceipt) error {
	return nil // Not implemented in basic gRPC version
}

// UploadMedia implements the ChannelAdapter interface
func (c *GRPCClient) UploadMedia(ctx context.Context, media *Media) (*MediaUpload, error) {
	return &MediaUpload{Success: false, Error: "not implemented"}, nil
}

// DownloadMedia implements the ChannelAdapter interface
func (c *GRPCClient) DownloadMedia(ctx context.Context, mediaID string) (*Media, error) {
	return nil, nil
}

// GetChannelType implements the ChannelAdapter interface
func (c *GRPCClient) GetChannelType() ChannelType {
	info := c.GetChannelInfo()
	return info.Type
}

// GetChannelInfo implements the ChannelAdapter interface
func (c *GRPCClient) GetChannelInfo() *ChannelInfo {
	resp, err := c.client.GetChannelInfo(context.Background(), &emptypb.Empty{})
	if err != nil {
		return &ChannelInfo{}
	}

	caps := &ChannelCapabilities{}
	if resp.Capabilities != nil {
		for _, ct := range resp.Capabilities.SupportedContentTypes {
			caps.SupportedContentTypes = append(caps.SupportedContentTypes, ContentType(ct))
		}
		caps.SupportsMedia = resp.Capabilities.SupportsMedia
		caps.SupportsLocation = resp.Capabilities.SupportsLocation
		caps.SupportsTemplates = resp.Capabilities.SupportsTemplates
		caps.SupportsInteractive = resp.Capabilities.SupportsInteractive
		caps.SupportsReadReceipts = resp.Capabilities.SupportsReadReceipts
		caps.MaxMessageLength = int(resp.Capabilities.MaxMessageLength)
		caps.MaxMediaSize = resp.Capabilities.MaxMediaSize
	}

	return &ChannelInfo{
		Type:         ChannelType(resp.Type),
		Name:         resp.Name,
		Description:  resp.Description,
		Version:      resp.Version,
		Author:       resp.Author,
		Capabilities: caps,
	}
}

// GetCapabilities implements the ChannelAdapter interface
func (c *GRPCClient) GetCapabilities() *ChannelCapabilities {
	info := c.GetChannelInfo()
	return info.Capabilities
}

// Proto message types (simplified - would normally be generated)

type InitializeRequest struct {
	Config map[string]string
}

type IsConnectedResponse struct {
	Connected bool
}

type SendMessageRequest struct {
	Id             string
	ConversationId string
	RecipientId    string
	ContentType    string
	Content        string
	Metadata       map[string]string
	Attachments    []*AttachmentProto
}

type AttachmentProto struct {
	Type         string
	Url          string
	Filename     string
	MimeType     string
	SizeBytes    int64
	ThumbnailUrl string
	Metadata     map[string]string
}

type SendResultResponse struct {
	Success    bool
	ExternalId string
	Status     string
	Error      string
	Timestamp  *timestamppb.Timestamp
}

type ChannelInfoResponse struct {
	Type         string
	Name         string
	Description  string
	Version      string
	Author       string
	Capabilities *CapabilitiesResponse
}

type CapabilitiesResponse struct {
	SupportedContentTypes []string
	SupportsMedia         bool
	SupportsLocation      bool
	SupportsTemplates     bool
	SupportsInteractive   bool
	SupportsReadReceipts  bool
	MaxMessageLength      int32
	MaxMediaSize          int64
}

// ChannelAdapterClient interface
type ChannelAdapterClient interface {
	Initialize(ctx context.Context, req *InitializeRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Connect(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Disconnect(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error)
	IsConnected(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*IsConnectedResponse, error)
	SendMessage(ctx context.Context, req *SendMessageRequest, opts ...grpc.CallOption) (*SendResultResponse, error)
	GetChannelInfo(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*ChannelInfoResponse, error)
}

// NewChannelAdapterClient creates a new client
func NewChannelAdapterClient(cc *grpc.ClientConn) ChannelAdapterClient {
	return &channelAdapterClient{cc: cc}
}

type channelAdapterClient struct {
	cc *grpc.ClientConn
}

func (c *channelAdapterClient) Initialize(ctx context.Context, req *InitializeRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (c *channelAdapterClient) Connect(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (c *channelAdapterClient) Disconnect(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (c *channelAdapterClient) IsConnected(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*IsConnectedResponse, error) {
	return &IsConnectedResponse{}, nil
}

func (c *channelAdapterClient) SendMessage(ctx context.Context, req *SendMessageRequest, opts ...grpc.CallOption) (*SendResultResponse, error) {
	return &SendResultResponse{Timestamp: timestamppb.New(time.Now())}, nil
}

func (c *channelAdapterClient) GetChannelInfo(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*ChannelInfoResponse, error) {
	return &ChannelInfoResponse{}, nil
}

// UnimplementedChannelAdapterServer for forward compatibility
type UnimplementedChannelAdapterServer struct{}

func (UnimplementedChannelAdapterServer) Initialize(context.Context, *InitializeRequest) (*emptypb.Empty, error) {
	return nil, nil
}
func (UnimplementedChannelAdapterServer) Connect(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}
func (UnimplementedChannelAdapterServer) Disconnect(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}
func (UnimplementedChannelAdapterServer) IsConnected(context.Context, *emptypb.Empty) (*IsConnectedResponse, error) {
	return nil, nil
}
func (UnimplementedChannelAdapterServer) SendMessage(context.Context, *SendMessageRequest) (*SendResultResponse, error) {
	return nil, nil
}
func (UnimplementedChannelAdapterServer) GetChannelInfo(context.Context, *emptypb.Empty) (*ChannelInfoResponse, error) {
	return nil, nil
}

// RegisterChannelAdapterServer registers the server
func RegisterChannelAdapterServer(s *grpc.Server, srv *GRPCServer) {
	// In production, this would register with the actual gRPC service descriptor
}
