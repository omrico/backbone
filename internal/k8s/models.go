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

type SecretResource struct {
	Password string
}

type RoleBindingResource struct {
	RoleRef string
	UserRef string
}

type RoleResourceDto struct {
	Roles []RoleResource `json:"roles"`
}
