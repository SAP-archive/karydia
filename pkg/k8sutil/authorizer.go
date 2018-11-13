package k8sutil

import (
	"k8s.io/apiserver/pkg/authorization/authorizer"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
)

func NewRBACAuthorizer(informerFactory kubeinformers.SharedInformerFactory) (authorizer.Authorizer, error) {
	return rbac.New(
		&rbac.RoleGetter{Lister: informerFactory.Rbac().V1().Roles().Lister()},
		&rbac.RoleBindingLister{Lister: informerFactory.Rbac().V1().RoleBindings().Lister()},
		&rbac.ClusterRoleGetter{Lister: informerFactory.Rbac().V1().ClusterRoles().Lister()},
		&rbac.ClusterRoleBindingLister{Lister: informerFactory.Rbac().V1().ClusterRoleBindings().Lister()},
	), nil
}
