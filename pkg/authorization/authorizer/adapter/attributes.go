package adapter

import (
	"strings"

	kapi "k8s.io/kubernetes/pkg/api"
	kauthorizer "k8s.io/kubernetes/pkg/auth/authorizer"
	"k8s.io/kubernetes/pkg/auth/user"

	oauthorizer "github.com/openshift/origin/pkg/authorization/authorizer"
)

// ensure we satisfy both interfaces
var _ = oauthorizer.AuthorizationAttributes(AdapterAttributes{})
var _ = kauthorizer.Attributes(AdapterAttributes{})

// AdapterAttributes satisfies both origin authorizer.AuthorizationAttributes and k8s authorizer.Attributes interfaces
type AdapterAttributes struct {
	namespace string
	name      string
	userName  string
	groups    []string
	oauthorizer.AuthorizationAttributes
}

// OriginAuthorizerAttributes adapts Kubernetes authorization attributes to Origin authorization attributes
// Note that some info (like resourceName, apiVersion, apiGroup) is not available from the Kubernetes attributes
func OriginAuthorizerAttributes(kattrs kauthorizer.Attributes) (kapi.Context, oauthorizer.AuthorizationAttributes) {
	// Build a context to hold the namespace and user info
	ctx := kapi.NewContext()
	ctx = kapi.WithNamespace(ctx, kattrs.GetNamespace())
	ctx = kapi.WithUser(ctx, &user.DefaultInfo{
		Name:   kattrs.GetUserName(),
		Groups: kattrs.GetGroups(),
	})

	// If the passed attributes already satisfy our interface, use it directly
	if oattrs, ok := kattrs.(oauthorizer.AuthorizationAttributes); ok {
		return ctx, oattrs
	}

	// Otherwise build what we can
	oattrs := &oauthorizer.DefaultAuthorizationAttributes{
		Verb:         kattrs.GetVerb(),
		APIGroup:     kattrs.GetAPIGroup(),
		APIVersion:   kattrs.GetAPIVersion(),
		Resource:     kattrs.GetResource(),
		ResourceName: kattrs.GetName(),

		NonResourceURL: kattrs.IsResourceRequest() == false,
		URL:            kattrs.GetPath(),

		// TODO: add to kube authorizer attributes
		// RequestAttributes interface{}
	}
	if len(kattrs.GetSubresource()) > 0 {
		oattrs.Resource = kattrs.GetResource() + "/" + kattrs.GetSubresource()
	}

	return ctx, oattrs
}

// KubernetesAuthorizerAttributes adapts Origin authorization attributes to Kubernetes authorization attributes
// The returned attributes can be passed to OriginAuthorizerAttributes to access extra information from the Origin attributes interface
func KubernetesAuthorizerAttributes(namespace string, userName string, groups []string, oattrs oauthorizer.AuthorizationAttributes) kauthorizer.Attributes {
	return AdapterAttributes{
		namespace:               namespace,
		userName:                userName,
		groups:                  groups,
		AuthorizationAttributes: oattrs,
	}
}

// GetNamespace satisfies the kubernetes authorizer.Attributes interface
// origin gets this value from the request context
func (a AdapterAttributes) GetNamespace() string {
	return a.namespace
}

func (a AdapterAttributes) GetName() string {
	return a.AuthorizationAttributes.GetResourceName()
}

func (a AdapterAttributes) GetSubresource() string {
	tokens := strings.SplitN(a.AuthorizationAttributes.GetResource(), "/", 2)
	if len(tokens) != 2 {
		return ""
	}
	return tokens[1]
}

func (a AdapterAttributes) GetResource() string {
	tokens := strings.SplitN(a.AuthorizationAttributes.GetResource(), "/", 2)
	if len(tokens) < 1 {
		return ""
	}
	return tokens[0]
}

// GetUserName satisfies the kubernetes authorizer.Attributes interface
// origin gets this value from the request context
func (a AdapterAttributes) GetUserName() string {
	return a.userName
}

// GetGroups satisfies the kubernetes authorizer.Attributes interface
// origin gets this value from the request context
func (a AdapterAttributes) GetGroups() []string {
	return a.groups
}

// IsReadOnly satisfies the kubernetes authorizer.Attributes interface based on the verb
func (a AdapterAttributes) IsReadOnly() bool {
	v := a.GetVerb()
	return v == "get" || v == "list" || v == "watch"
}

// IsResourceRequest satisfies the kubernetes authorizer.Attributes interface
func (a AdapterAttributes) IsResourceRequest() bool {
	return !a.IsNonResourceURL()
}

// GetPath satisfies the kubernetes authorizer.Attributes interface
func (a AdapterAttributes) GetPath() string {
	return a.GetURL()
}
