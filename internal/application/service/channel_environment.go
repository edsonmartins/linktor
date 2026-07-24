package service

import (
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// CredentialEnvironmentKey is the declarative credential tag that binds a
// credential set to an environment. There is no way to infer from a token
// whether it is a production credential — credentials are opaque and encrypted
// — so the check is declarative by design: the caller states which environment
// the credentials belong to and the service enforces consistency. The hard
// guarantee against synthetic traffic reaching real recipients is the sandbox
// delivery allowlist, not this validation.
const CredentialEnvironmentKey = "credential_environment"

// SandboxTestPhoneNumberIDsKey is the Config key holding the tenant-declared
// comma-separated list of Meta test phone_number_ids a sandbox
// whatsapp_official channel may use. Declarative, like the credential tag: it
// is not verified against the Graph API in this phase.
const SandboxTestPhoneNumberIDsKey = "sandbox_test_phone_number_ids"

// environmentBindingDetails tags a validation error as coming from the
// environment/credential binding rules so the handler can audit the rejected
// attempt specifically (INV-023): trying to cross environments is a security
// signal worth a trail entry, unlike ordinary validation noise.
var environmentBindingDetails = map[string]string{"validation": "environment_binding"}

// IsEnvironmentBindingError reports whether err is a rejection from the
// environment/credential binding or immutability validations.
func IsEnvironmentBindingError(err error) bool {
	appErr := errors.GetAppError(err)
	return appErr != nil && appErr.Details["validation"] == "environment_binding"
}

func environmentBindingError(msg string) error {
	return errors.New(errors.ErrCodeValidation, msg).WithDetails(environmentBindingDetails)
}

// validateChannelEnvironmentBinding enforces INV-002 at the domain edge:
// production credentials cannot be bound to a sandbox channel and vice versa.
// It runs on creation and on every update (over the post-merge config and
// credentials), so a later credential edit cannot quietly cross environments.
func validateChannelEnvironmentBinding(env entity.ChannelEnvironment, chType entity.ChannelType, config, credentials map[string]string) error {
	declared := credentials[CredentialEnvironmentKey]

	if env == entity.ChannelEnvironmentSandbox {
		if declared != string(entity.ChannelEnvironmentSandbox) {
			return environmentBindingError(
				"sandbox channel requires credentials[\"" + CredentialEnvironmentKey + "\"] = \"sandbox\": declare explicitly that these are test credentials")
		}
		if chType == entity.ChannelTypeWhatsAppOfficial {
			return validateSandboxPhoneNumberID(config)
		}
		return nil
	}

	if declared == string(entity.ChannelEnvironmentSandbox) {
		return environmentBindingError(
			"credentials declared as sandbox cannot be bound to a production channel")
	}
	return nil
}

// validateSandboxPhoneNumberID requires a sandbox whatsapp_official channel to
// declare its list of test phone_number_ids and to use one of them. Both the
// id and the list are required: a sandbox channel without a declared test
// number list is indistinguishable from one pointing at a production number,
// so it fails closed.
func validateSandboxPhoneNumberID(config map[string]string) error {
	phoneNumberID := strings.TrimSpace(config["phone_number_id"])
	if phoneNumberID == "" {
		return environmentBindingError(
			"sandbox whatsapp_official channel requires config[\"phone_number_id\"]")
	}
	declared := config[SandboxTestPhoneNumberIDsKey]
	if strings.TrimSpace(declared) == "" {
		return environmentBindingError(
			"sandbox whatsapp_official channel requires config[\"" + SandboxTestPhoneNumberIDsKey + "\"] listing the tenant's Meta test phone_number_ids")
	}
	for _, id := range strings.Split(declared, ",") {
		if strings.TrimSpace(id) == phoneNumberID {
			return nil
		}
	}
	return environmentBindingError(
		"phone_number_id is not in the declared " + SandboxTestPhoneNumberIDsKey + " list for this sandbox channel")
}
