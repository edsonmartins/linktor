package database

import "github.com/msgfy/linktor/internal/domain/entity"

// sensitiveConfigKeys is the single source of truth for Channel.Config keys whose
// values are secrets: encrypted at rest here and redacted from API responses in
// entity.Channel.MarshalJSON. Non-listed keys (phone_number_id, waba_id, proxy
// host/port, advanced settings, ...) stay plaintext so they remain queryable via
// `config->>'key'`. The Credentials map is encrypted in full and is not listed.
var sensitiveConfigKeys = entity.SensitiveConfigKeys

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
