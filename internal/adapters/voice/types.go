package voice

import (
	"time"
)

// CallDirection represents the direction of a call
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// CallStatus represents the status of a call
type CallStatus string

const (
	CallStatusInitiated  CallStatus = "initiated"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusAnswered   CallStatus = "answered"
	CallStatusInProgress CallStatus = "in-progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusBusy       CallStatus = "busy"
	CallStatusNoAnswer   CallStatus = "no-answer"
	CallStatusFailed     CallStatus = "failed"
	CallStatusCanceled   CallStatus = "canceled"
)

// Call represents a voice call
type Call struct {
	ID            string            `json:"id"`
	ExternalID    string            `json:"externalId"`
	ChannelID     string            `json:"channelId"`
	Direction     CallDirection     `json:"direction"`
	Status        CallStatus        `json:"status"`
	From          string            `json:"from"`
	To            string            `json:"to"`
	CallerName    string            `json:"callerName,omitempty"`
	Duration      int               `json:"duration"` // in seconds
	RecordingURL  string            `json:"recordingUrl,omitempty"`
	Transcription string            `json:"transcription,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	StartedAt     time.Time         `json:"startedAt"`
	AnsweredAt    *time.Time        `json:"answeredAt,omitempty"`
	EndedAt       *time.Time        `json:"endedAt,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// MakeCallInput represents input for making a call
type MakeCallInput struct {
	To           string            `json:"to"`
	From         string            `json:"from,omitempty"`
	CallbackURL  string            `json:"callbackUrl,omitempty"`
	StatusURL    string            `json:"statusUrl,omitempty"`
	Record       bool              `json:"record,omitempty"`
	Transcribe   bool              `json:"transcribe,omitempty"`
	Timeout      int               `json:"timeout,omitempty"` // ring timeout in seconds
	MachineDetection bool          `json:"machineDetection,omitempty"`
	TwiML        string            `json:"twiml,omitempty"` // For Twilio
	NCCO         []interface{}     `json:"ncco,omitempty"`  // For Vonage
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// MakeCallResult represents the result of making a call
type MakeCallResult struct {
	CallID     string     `json:"callId"`
	ExternalID string     `json:"externalId"`
	Status     CallStatus `json:"status"`
	Message    string     `json:"message,omitempty"`
}

// IVRAction represents an action in an IVR flow
type IVRAction interface {
	ActionType() string
}

// IVRSay represents a text-to-speech action
type IVRSay struct {
	Text     string `json:"text"`
	Voice    string `json:"voice,omitempty"`    // e.g., "alice", "man", "woman"
	Language string `json:"language,omitempty"` // e.g., "pt-BR", "en-US"
	Loop     int    `json:"loop,omitempty"`     // number of times to repeat
}

func (a IVRSay) ActionType() string { return "say" }

// IVRPlay represents an audio file playback action
type IVRPlay struct {
	URL   string `json:"url"`
	Loop  int    `json:"loop,omitempty"`
	Digits string `json:"digits,omitempty"` // DTMF tones to play
}

func (a IVRPlay) ActionType() string { return "play" }

// IVRGather represents a DTMF/speech input collection action
type IVRGather struct {
	Input       []string    `json:"input"`       // ["dtmf", "speech"]
	Timeout     int         `json:"timeout"`     // seconds to wait
	NumDigits   int         `json:"numDigits,omitempty"`
	FinishOnKey string      `json:"finishOnKey,omitempty"` // e.g., "#"
	ActionURL   string      `json:"actionUrl,omitempty"`
	Method      string      `json:"method,omitempty"` // GET or POST
	Language    string      `json:"language,omitempty"`
	Hints       []string    `json:"hints,omitempty"` // speech recognition hints
	Nested      []IVRAction `json:"nested,omitempty"`
}

func (a IVRGather) ActionType() string { return "gather" }

// IVRRecord represents a recording action
type IVRRecord struct {
	MaxLength    int    `json:"maxLength,omitempty"`    // max recording length in seconds
	Timeout      int    `json:"timeout,omitempty"`      // silence timeout
	FinishOnKey  string `json:"finishOnKey,omitempty"`
	Transcribe   bool   `json:"transcribe,omitempty"`
	PlayBeep     bool   `json:"playBeep,omitempty"`
	ActionURL    string `json:"actionUrl,omitempty"`
	RecordingURL string `json:"recordingUrl,omitempty"`
}

func (a IVRRecord) ActionType() string { return "record" }

// IVRDial represents a dial/transfer action
type IVRDial struct {
	Number      string `json:"number,omitempty"`
	SIPEndpoint string `json:"sipEndpoint,omitempty"`
	Queue       string `json:"queue,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
	CallerID    string `json:"callerId,omitempty"`
	Record      bool   `json:"record,omitempty"`
	ActionURL   string `json:"actionUrl,omitempty"`
}

func (a IVRDial) ActionType() string { return "dial" }

// IVRHangup represents a hangup action
type IVRHangup struct {
	Reason string `json:"reason,omitempty"`
}

func (a IVRHangup) ActionType() string { return "hangup" }

// IVRPause represents a pause/wait action
type IVRPause struct {
	Length int `json:"length"` // seconds
}

func (a IVRPause) ActionType() string { return "pause" }

// IVRRedirect represents a redirect to another URL
type IVRRedirect struct {
	URL    string `json:"url"`
	Method string `json:"method,omitempty"`
}

func (a IVRRedirect) ActionType() string { return "redirect" }

// IVRQueue represents a queue action
type IVRQueue struct {
	Name       string `json:"name"`
	WaitURL    string `json:"waitUrl,omitempty"`
	ActionURL  string `json:"actionUrl,omitempty"`
	MaxWait    int    `json:"maxWait,omitempty"` // max wait time in seconds
}

func (a IVRQueue) ActionType() string { return "queue" }

// IVRConference represents a conference action
type IVRConference struct {
	Name          string `json:"name"`
	Muted         bool   `json:"muted,omitempty"`
	StartOnEnter  bool   `json:"startOnEnter,omitempty"`
	EndOnExit     bool   `json:"endOnExit,omitempty"`
	WaitURL       string `json:"waitUrl,omitempty"`
	MaxParticipants int  `json:"maxParticipants,omitempty"`
	Record        bool   `json:"record,omitempty"`
}

func (a IVRConference) ActionType() string { return "conference" }

// WebhookEvent represents an incoming webhook event
type WebhookEvent struct {
	Type        string            `json:"type"`
	CallID      string            `json:"callId"`
	ExternalID  string            `json:"externalId"`
	Status      CallStatus        `json:"status"`
	Direction   CallDirection     `json:"direction"`
	From        string            `json:"from"`
	To          string            `json:"to"`
	Duration    int               `json:"duration,omitempty"`
	Digits      string            `json:"digits,omitempty"`      // DTMF input
	SpeechResult string           `json:"speechResult,omitempty"` // Speech recognition result
	RecordingURL string           `json:"recordingUrl,omitempty"`
	Transcription string          `json:"transcription,omitempty"`
	Error       string            `json:"error,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	RawPayload  map[string]interface{} `json:"rawPayload,omitempty"`
}

// VoiceConfig represents voice channel configuration
type VoiceConfig struct {
	Provider      string            `json:"provider"` // twilio, vonage, amazon_connect, asterisk, freeswitch
	PhoneNumber   string            `json:"phoneNumber"`
	WebhookURL    string            `json:"webhookUrl"`
	StatusURL     string            `json:"statusUrl"`
	RecordCalls   bool              `json:"recordCalls"`
	TranscribeCalls bool            `json:"transcribeCalls"`
	DefaultVoice  string            `json:"defaultVoice"`
	DefaultLanguage string          `json:"defaultLanguage"`
	Credentials   map[string]string `json:"credentials"`
}

// ProviderCapabilities represents what a voice provider supports
type ProviderCapabilities struct {
	OutboundCalls    bool `json:"outboundCalls"`
	InboundCalls     bool `json:"inboundCalls"`
	Recording        bool `json:"recording"`
	Transcription    bool `json:"transcription"`
	TextToSpeech     bool `json:"textToSpeech"`
	SpeechRecognition bool `json:"speechRecognition"`
	DTMF             bool `json:"dtmf"`
	Conferencing     bool `json:"conferencing"`
	CallQueues       bool `json:"callQueues"`
	SIP              bool `json:"sip"`
	WebRTC           bool `json:"webrtc"`
}

// Recording represents a call recording
type Recording struct {
	ID           string    `json:"id"`
	CallID       string    `json:"callId"`
	URL          string    `json:"url"`
	Duration     int       `json:"duration"`
	Size         int64     `json:"size"`
	Format       string    `json:"format"` // mp3, wav
	Transcription string   `json:"transcription,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// TranscriptionResult represents the result of speech transcription
type TranscriptionResult struct {
	CallID      string    `json:"callId"`
	RecordingID string    `json:"recordingId"`
	Text        string    `json:"text"`
	Confidence  float64   `json:"confidence"`
	Language    string    `json:"language"`
	Words       []Word    `json:"words,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Word represents a transcribed word with timing
type Word struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"` // seconds
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence"`
}
