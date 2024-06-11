package k8s

type UserResource struct {
	Email     string
	FirstName string
	LastName  string
}

type UserPasswordResource struct {
	UserResource
	SecretRef string
}

type RoleResource struct {
	RoleName    string   `json:"roleName"`
	Permissions []string `json:"permissions"`
}

type SecretPasswordResource struct {
	Password string
}

type SecretJwtKeysResource struct {
	PrivateKey string `json:"private.key"`
	PublicKey  string `json:"public.key"`
}

type RoleBindingResource struct {
	RoleRef string
	UserRef string
}

type RoleResourceDto struct {
	Roles []RoleResource `json:"roles"`
}

type ConfigResource struct {
	Mode                string
	SyncIntervalSeconds int
	CookieStoreKeyRef   string
	Oidc                struct {
		EncryptionKeyRef  string // authz code flow state enc
		JwtSigningKeysRef string // token sign
		Providers         []struct {
			ProviderName    string
			ProviderType    string
			ProviderUrl     string
			ClientID        string `json:"clientId"`
			ClientSecretRef string
		}
	}
}
