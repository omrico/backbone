package auth

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/omrico/backbone/internal/k8s"
)

type UserAuth struct {
	Username      string              `json:"username"`
	RolesAndPerms map[string][]string `json:"roles"`
	Expiration    int64               `json:"expiration"`
}

func BuildAuthFromCtx(ctx *gin.Context) (*UserAuth, error) {
	username := ctx.GetString("username")
	if username == "" {
		return &UserAuth{}, errors.New("unable to get username from context")
	}
	roles := ctx.GetString("roles")
	var rolesDTO k8s.RoleResourceDto
	err := json.Unmarshal([]byte(roles), &rolesDTO)
	if err != nil {
		return &UserAuth{}, errors.New("unable to get roles from context")
	}
	roleAndPermsMap := map[string][]string{}
	for _, r := range rolesDTO.Roles {
		roleAndPermsMap[r.RoleName] = r.Permissions
	}
	expiration := ctx.GetInt64("exp")

	return &UserAuth{
		Username:      username,
		RolesAndPerms: roleAndPermsMap,
		Expiration:    expiration,
	}, nil
}
