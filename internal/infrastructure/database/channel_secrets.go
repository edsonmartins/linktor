package database

import "github.com/msgfy/linktor/internal/domain/entity"

// sensitiveConfigKeys lists Channel.Config keys whose values are secrets and
// must be encrypted at rest. Non-listed keys (phone_number_id, waba_id, proxy
// host/port, advanced settings, ...) stay plaintext so they remain queryable
// via `config->>'key'` and readable for non-secret behavior.
//
// The Credentials map is encrypted in full and is therefore not listed here.
var sensitiveConfigKeys = map[string]bool{
	"access_token":          true,
	"user_access_token":     true,
	"page_access_token":     true,
	"auth_token":            true,
	"bot_token":             true,
	"app_secret":            true,
	"api_secret":            true,
	"api_key":               true,
	"api_key_secret":        true,
	"api_key_sid":           true,
	"account_sid":           true,
	"messaging_service_sid": true,
	"webhook_secret":        true,
	"verify_token":          true,
	"smtp_password":         true,
	"imap_password":         true,
	"mailgun_api_key":       true,
	"sendgrid_api_key":      true,
	"postmark_server_token": true,
	"ses_access_key_id":     true,
	"ses_secret_key":        true,
	"private_key":           true,
	"secret":                true,
	"token":                 true,
	"password":              true,
}

// encryptChannelSecrets returns encrypted copies of the channel's Credentials
// (all values) and Config (sensitive keys only), without mutating the channel.
// When the repository has no encryptor configured the originals are returned
// unchanged.
func (r *ChannelRepository) encryptChannelSecrets(ch *entity.Channel) (creds, config map[string]string, err error) {
	if r.enc == nil {
		return ch.Credentials, ch.Config, nil
	}

	creds, err = r.enc.EncryptMap(ch.Credentials)
	if err != nil {
		return nil, nil, err
	}

	config, err = r.enc.EncryptKeys(ch.Config, sensitiveConfigKeys)
	if err != nil {
		return nil, nil, err
	}

	return creds, config, nil
}

// decryptChannelSecrets decrypts the channel's Credentials and Config in place
// after a row scan. Values that were stored before encryption was enabled lack
// the enc prefix and pass through untouched.
func (r *ChannelRepository) decryptChannelSecrets(ch *entity.Channel) error {
	if r.enc == nil {
		return nil
	}

	creds, err := r.enc.DecryptMap(ch.Credentials)
	if err != nil {
		return err
	}
	ch.Credentials = creds

	config, err := r.enc.DecryptKeys(ch.Config)
	if err != nil {
		return err
	}
	ch.Config = config

	return nil
}
