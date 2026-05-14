package rbac

import (
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

type Authorizer struct {
	enforcer *casbin.Enforcer
}

func New() (*Authorizer, error) {
	text := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (p.act == "*" || p.act == r.act)
`
	m, err := model.NewModelFromString(text)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}
	return &Authorizer{enforcer: enforcer}, nil
}

func (a *Authorizer) AddRoleForUser(userID, role string) error {
	_, err := a.enforcer.AddRoleForUser(userID, role)
	return err
}

func (a *Authorizer) DeleteRoleForUser(userID, role string) error {
	_, err := a.enforcer.DeleteRoleForUser(userID, role)
	return err
}

func (a *Authorizer) SetRolePermissions(role string, permissions []string) error {
	policies, err := a.enforcer.GetFilteredPolicy(0, role)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		args := make([]interface{}, len(policy))
		for i, value := range policy {
			args[i] = value
		}
		_, _ = a.enforcer.RemovePolicy(args...)
	}
	for _, permission := range permissions {
		resource, action, ok := strings.Cut(permission, ":")
		if !ok {
			resource = permission
			action = "*"
		}
		if _, err := a.enforcer.AddPolicy(role, resource, action); err != nil {
			return err
		}
	}
	return nil
}

func (a *Authorizer) Can(userID, resource, action string) bool {
	allowed, err := a.enforcer.Enforce(userID, resource, action)
	return err == nil && allowed
}
