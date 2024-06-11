package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/omrico/backbone/internal/config"
	"github.com/omrico/backbone/internal/misc"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	userMapping     map[string]UserResource   // email to user data
	roleMapping     map[string]RoleResource   // roleName to role data
	userRoleMapping map[string][]RoleResource // user to role(s)
	k8sDynClient    *dynamic.DynamicClient
	Cfg             *config.Config
}

func (c *Client) StartSync(wg *sync.WaitGroup) {
	wg.Wait()
	logger := misc.GetLogger()
	logger.Infof("config ready, starting k8s sync client. refresh will happen every %d seconds", c.Cfg.SyncInterval)
	logger.Info("reading data from k8s... (first run)")
	c.ReadDataFromK8s()
	ticker := time.NewTicker(time.Second * time.Duration(c.Cfg.SyncInterval))
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				logger.Info("reading data from k8s...")
				c.ReadDataFromK8s()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *Client) NewClient() {
	logger := misc.GetLogger()
	kubeconfig := c.Cfg.KubeConfig
	var restconfig *rest.Config
	var dynamicClient *dynamic.DynamicClient
	var err error

	if c.k8sDynClient == nil {
		if kubeconfig != "" {
			logger.Info("KUBECONFIG set, assuming not in cluster mode")

			restconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				logger.Fatalf("error loading kubeconfig: %v\n", err)
				return
			}
		} else {
			logger.Info("KUBECONFIG not set, assuming in cluster mode")

			restconfig, err = rest.InClusterConfig()
			if err != nil {
				logger.Fatalf("error creating in-cluster config: %v\n", err)
				return
			}
		}

		// Create dynamic client for CRD resources
		dynamicClient, err = dynamic.NewForConfig(restconfig)
		if err != nil {
			logger.Fatalf("error creating dynamic client: %+v", err)
			os.Exit(1)
		}

		c.k8sDynClient = dynamicClient
	}
}

func (client *Client) ReadDataFromK8s() {
	logger := misc.GetLogger()

	var err error

	// build user map
	crs, err := getCRsForGroupKind("iam-backbone.org", "backboneuser", client.k8sDynClient)
	if err != nil {
		logger.Errorf("error fetching backboneuser CRs: %+v", err)
	}
	client.userMapping = buildUserMapping(crs)

	// build user-role map
	// fetch roles
	crs, err = getCRsForGroupKind("iam-backbone.org", "backbonerole", client.k8sDynClient)
	if err != nil {
		logger.Errorf("error fetching backboneroles CRs: %+v", err)
	}
	client.roleMapping = buildRoleMapping(crs)
	crs, err = getCRsForGroupKind("iam-backbone.org", "backbonerolebinding", client.k8sDynClient)
	if err != nil {
		logger.Errorf("error fetching backbonerolebindings CRs: %+v", err)
	}
	client.userRoleMapping = buildUserRoleMapping(crs, client.k8sDynClient)
}

func buildRoleMapping(crs *unstructured.UnstructuredList) map[string]RoleResource {
	var rolemap = map[string]RoleResource{}
	for _, cr := range crs.Items {
		specData, _ := json.Marshal(cr.Object["spec"])
		var spec RoleResource
		_ = json.Unmarshal(specData, &spec)
		rolemap[spec.RoleName] = spec
	}
	return rolemap
}

func buildUserRoleMapping(crs *unstructured.UnstructuredList, dynClient *dynamic.DynamicClient) map[string][]RoleResource {
	var userRolesMap = map[string][]RoleResource{}
	for _, cr := range crs.Items {
		// extract rolebinbding data
		specData, _ := json.Marshal(cr.Object["spec"])
		var spec RoleBindingResource
		_ = json.Unmarshal(specData, &spec)

		// extract user data
		userCR, _ := getCRForGroupKind("iam-backbone.org", "backboneuser", spec.UserRef, dynClient)
		specData, _ = json.Marshal(userCR.Object["spec"])
		var user UserResource
		_ = json.Unmarshal(specData, &user)

		// extract the role
		roleCR, _ := getCRForGroupKind("iam-backbone.org", "backbonerole", spec.RoleRef, dynClient)
		specData, _ = json.Marshal(roleCR.Object["spec"])
		var role RoleResource
		_ = json.Unmarshal(specData, &role)

		// append role to user roles
		currRoles := userRolesMap[user.Email]
		userRolesMap[user.Email] = append(currRoles, role)
	}
	return userRolesMap
}

func buildUserMapping(crs *unstructured.UnstructuredList) map[string]UserResource {
	var usermap = map[string]UserResource{}
	for _, cr := range crs.Items {
		specData, _ := json.Marshal(cr.Object["spec"])
		var spec UserResource
		_ = json.Unmarshal(specData, &spec)
		usermap[spec.Email] = spec
	}
	return usermap
}

func getCRsForGroupKind(group string, kind string, dynClient *dynamic.DynamicClient) (*unstructured.UnstructuredList, error) {
	// Retrieve the resource schema for the given CRD
	resource := schema.GroupVersionResource{
		Group:    group,
		Version:  "v1",
		Resource: kind + "s", // Plural form of the kind
	}

	// Fetch the list of CRs
	crs, err := dynClient.Resource(resource).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return &unstructured.UnstructuredList{}, err
	}

	return crs, nil
}

func getCRForGroupKind(group string, kind string, crName string, dynClient *dynamic.DynamicClient) (*unstructured.Unstructured, error) {
	// Retrieve the resource schema for the given CRD
	resource := schema.GroupVersionResource{
		Group:    group,
		Version:  "v1",
		Resource: kind + "s", // Plural form of the kind
	}

	// @TODO  - add config to handle namespace
	// Fetch the list of CRs
	cr, err := dynClient.Resource(resource).Namespace("default").Get(context.Background(), crName, v1.GetOptions{})
	if err != nil {
		return &unstructured.Unstructured{}, err
	}
	return cr, nil
}

func getCRForGroupKindWithLabel(group, kind, labelName, labelValue string, dynClient *dynamic.DynamicClient) (*unstructured.Unstructured, error) {
	// Retrieve the resource schema for the given CRD
	resource := schema.GroupVersionResource{
		Group:    group,
		Version:  "v1",
		Resource: kind + "s", // Plural form of the kind
	}

	// Label selector to filter CRs
	labelSelector := labels.SelectorFromSet(labels.Set{labelName: labelValue})

	// Fetch the list of CRs and filter by username
	crs, err := dynClient.Resource(resource).Namespace("default").List(context.Background(), v1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	if len(crs.Items) == 0 {
		return nil, fmt.Errorf("no CR With label %s and value %s found", labelName, labelValue)
	}

	return &crs.Items[0], nil
}

func (client *Client) GetUser(email string) (UserResource, error) {
	user, ok := client.userMapping[email]
	if !ok {
		return UserResource{}, errors.New("user not found")
	}
	return user, nil
}

func getUserWithPasswordByUsername(username string, dynClient *dynamic.DynamicClient) (UserPasswordResource, error) {
	crs, _ := getCRsForGroupKind("iam-backbone.org", "backboneuser", dynClient)
	for _, cr := range crs.Items {
		specData, _ := json.Marshal(cr.Object["spec"])
		var userWithPassword UserPasswordResource
		_ = json.Unmarshal(specData, &userWithPassword)
		if userWithPassword.Email == username {
			return userWithPassword, nil
		}
	}

	return UserPasswordResource{}, errors.New("user not found")
}

func (client *Client) AssertPassword(username string, password string) bool {
	logger := misc.GetLogger()
	userWithPassword, err := getUserWithPasswordByUsername(username, client.k8sDynClient)
	if err != nil {
		logger.Warnf("could not get user by username: %s", err.Error())
		return false
	}

	secretCR, _ := getCRForGroupKind("", "secret", userWithPassword.SecretRef, client.k8sDynClient)
	specData, _ := json.Marshal(secretCR.Object["data"])
	var k8sPw SecretPasswordResource
	_ = json.Unmarshal(specData, &k8sPw)

	pw, err := base64.StdEncoding.DecodeString(k8sPw.Password)
	if err != nil {
		logger.Warnf("could not decode password of user %s", username)
		return false
	}
	return password == string(pw)
}

func (client *Client) GetUserRoles(email string) ([]RoleResource, error) {
	roles, ok := client.userRoleMapping[email]
	if !ok {
		return []RoleResource{}, errors.New("user not found")
	}
	return roles, nil
}

// fetchSecretFromRef Abstracts fetching and rendering a secret for a given string ref
func (c *Client) fetchSecretFromRef(secretRef string) (string, error) {
	logger := misc.GetLogger()
	secretCR, err := getCRForGroupKind("", "secret", secretRef, c.k8sDynClient)
	if err != nil {
		logger.Errorf("failed fetching secret resource for %s: %s", secretRef, err.Error())
		return "", errors.New("failed fetching secret resource")
	}
	specData, err := json.Marshal(secretCR.Object["data"])
	if err != nil {
		logger.Errorf("failed marshalling secret for %s: %s", secretRef, err.Error())
		return "", errors.New("failed marshalling secret")
	}
	var k8sPw SecretPasswordResource
	err = json.Unmarshal(specData, &k8sPw)
	if err != nil {
		logger.Errorf("failed unmarshalling secret for %s: %s", secretRef, err.Error())
		return "", errors.New("failed unmarshalling secret")
	}
	secret, err := base64.StdEncoding.DecodeString(k8sPw.Password)
	if err != nil {
		logger.Errorf("failed decoding secret for %s: %s", secretRef, err.Error())
		return "", errors.New("failed decoding secret")
	}
	return string(secret), nil
}

// fetchJwtKeysSecretFromRef fetches and renders the pub/private JWT keys
func (c *Client) fetchJwtKeysSecretFromRef(secretRef string) (string, string, error) {
	logger := misc.GetLogger()
	secretCR, err := getCRForGroupKind("", "secret", secretRef, c.k8sDynClient)
	if err != nil {
		logger.Errorf("failed fetching secret resource for %s: %s", secretRef, err.Error())
		return "", "", errors.New("failed fetching jwt keys secret resource")
	}
	specData, err := json.Marshal(secretCR.Object["data"])
	if err != nil {
		logger.Errorf("failed marshalling jwt keys secret for %s: %s", secretRef, err.Error())
		return "", "", errors.New("failed marshalling jwt kets secret")
	}
	var k8sPw SecretJwtKeysResource
	err = json.Unmarshal(specData, &k8sPw)
	if err != nil {
		logger.Errorf("failed unmarshalling jwt keys secret for %s: %s", secretRef, err.Error())
		return "", "", errors.New("failed unmarshalling jwt keys secret")
	}
	publicKey, err := base64.StdEncoding.DecodeString(k8sPw.PublicKey)
	if err != nil {
		logger.Errorf("failed decoding public key secret for %s: %s", secretRef, err.Error())
		return "", "", errors.New("failed decoding public key secret")
	}

	privateKey, err := base64.StdEncoding.DecodeString(k8sPw.PrivateKey)
	if err != nil {
		logger.Errorf("failed decoding private key secret for %s: %s", secretRef, err.Error())
		return "", "", errors.New("failed decoding private key secret")
	}

	return string(privateKey), string(publicKey), nil
}

func (c *Client) ConfigWithWatcher(gcfg *config.Config, wg *sync.WaitGroup) {
	logger := misc.GetLogger()
	logger.Info("starting watcher - read config CR")
	resource := schema.GroupVersionResource{
		Group:    "iam-backbone.org",
		Version:  "v1",
		Resource: "backboneconfigs",
	}
	watcher, err := c.k8sDynClient.Resource(resource).Namespace("default").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		logger.Warnf("Error watching CRs: %v\n", err)
		return
	}
	firstRun := true

	// Define a function to handle events
	eventHandler := func(obj interface{}) {
		// Convert the event object to a CR
		cr, ok := obj.(*unstructured.Unstructured)
		if !ok {
			logger.Info("error converting CR")
			return
		}

		specData, _ := json.Marshal(cr.Object["spec"])
		var cfg ConfigResource
		_ = json.Unmarshal(specData, &cfg)

		gcfg.Mode = cfg.Mode
		gcfg.SyncInterval = cfg.SyncIntervalSeconds

		if gcfg.Mode == "SESSIONS" {
			gcfg.CookieStoreKey, err = c.fetchSecretFromRef(cfg.CookieStoreKeyRef)
		}

		if gcfg.Mode == "OIDC_BROKER" {
			var providerSlice []config.ProviderConfig
			for _, prov := range cfg.Oidc.Providers {
				clientSecret, err := c.fetchSecretFromRef(prov.ClientSecretRef)
				if err != nil {
					logger.Warnf("cannot fetch oidc client secret from config, skipping provider %s", prov.ProviderName)
					continue
				}
				providerConfig := config.ProviderConfig{
					ProviderName: prov.ProviderName,
					ProviderType: prov.ProviderType,
					ProviderUrl:  prov.ProviderUrl,
					ClientID:     prov.ClientID,
					ClientSecret: clientSecret,
				}
				providerSlice = append(providerSlice, providerConfig)
			}

			encKey, err := c.fetchSecretFromRef(cfg.Oidc.EncryptionKeyRef)
			if err != nil {
				logger.Error("cannot fetch oidc enc key from config")
				return
			}

			privateKey, publicKey, err := c.fetchJwtKeysSecretFromRef(cfg.Oidc.JwtSigningKeysRef)
			if err != nil {
				logger.Errorf("failed getting private and public keys: %s", err)
				return
			}
			gcfg.Oidc = config.OidcConfig{
				EncryptionKey: encKey,
				PublicKey:     publicKey,
				PrivateKey:    privateKey,
				Providers:     providerSlice,
			}
		}

		// signal that the config is ready on boot
		if firstRun {
			firstRun = false
			wg.Done()
		}
	}

	// Start watching for events
	stopCh := make(chan struct{})
	defer close(stopCh)

	go func() {
		for {
			event, ok := <-watcher.ResultChan()
			if !ok {
				logger.Info("Watcher channel closed")
				return
			}
			logger.Infof("event received: %+v", event)
			eventHandler(event.Object)
		}
	}()
}
