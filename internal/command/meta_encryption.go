package command

import (
	"fmt"
	"os"

	"github.com/opentofu/opentofu/internal/configs"
	"github.com/opentofu/opentofu/internal/encryption"
	"github.com/opentofu/opentofu/internal/encryption/config"
	"github.com/opentofu/opentofu/internal/encryption/keyprovider/static"
	"github.com/opentofu/opentofu/internal/encryption/method/aesgcm"
	"github.com/opentofu/opentofu/internal/encryption/registry/lockingencryptionregistry"
	"github.com/opentofu/opentofu/internal/tfdiags"
)

const encryptionConfigEnvName = "TF_ENCRYPTION"

func (m *Meta) Encryption() (encryption.Encryption, tfdiags.Diagnostics) {
	path, err := os.Getwd()
	if err != nil {
		return nil, tfdiags.Diagnostics{}.Append(fmt.Errorf("Error getting pwd: %w", err))
	}

	return m.EncryptionFromPath(path)
}

func (m *Meta) EncryptionFromPath(path string) (encryption.Encryption, tfdiags.Diagnostics) {
	// This is not ideal, but given how fragmented the command package is, loading the root module here is our best option
	// See other meta commands like version check which do that same.
	module, diags := m.loadSingleModule(path)
	if diags.HasErrors() {
		return nil, diags
	}
	enc, encDiags := m.EncryptionFromModule(module)
	diags = diags.Append(encDiags)
	return enc, diags
}

func (m *Meta) EncryptionFromModule(module *configs.Module) (encryption.Encryption, tfdiags.Diagnostics) {
	reg := lockingencryptionregistry.New()
	if err := reg.RegisterKeyProvider(static.New()); err != nil {
		panic(err)
	}
	if err := reg.RegisterMethod(aesgcm.New()); err != nil {
		panic(err)
	}

	cfg := module.Encryption
	var diags tfdiags.Diagnostics

	env := os.Getenv(encryptionConfigEnvName)
	if len(env) != 0 {
		envCfg, envDiags := config.LoadConfigFromString(encryptionConfigEnvName, env)
		diags = diags.Append(envDiags)
		if envDiags.HasErrors() {
			return nil, diags
		}
		cfg = cfg.Merge(envCfg)
	}

	enc, encDiags := encryption.New(reg, cfg)
	diags = diags.Append(encDiags)

	return enc, diags
}
