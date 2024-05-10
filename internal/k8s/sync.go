package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
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

func getCRForGroupKindWithLabel(group string, kind string, labelName string, labelValue string, dynClient *dynamic.DynamicClient) (*unstructured.Unstructured, error) {
	// Retrieve the resource schema for the given CRD
	resource := schema.GroupVersionResource{
		Group:    group,
		Version:  "v1",
		Resource: kind + "s", // Plural form of the kind
	}

	// Label selector to filter CRs
	labelSelector := labels.SelectorFromSet(labels.Set{labelName: labelValue})

	// Fetch the list of CRs with the specified label
	crs, err := dynClient.Resource(resource).Namespace("default").List(context.Background(), v1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		return &unstructured.Unstructured{}, err
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

func (client *Client) AssertPassword(username string, password string) bool {
	logger := misc.GetLogger()
	username = strings.Replace(username, "@", "__at__", 1)
	userCR, _ := getCRForGroupKindWithLabel("iam-backbone.org", "backboneuser", "email", username, client.k8sDynClient)
	specData, _ := json.Marshal(userCR.Object["spec"])
	var userWithPassword UserPasswordResource
	_ = json.Unmarshal(specData, &userWithPassword)

	//
	secretCR, _ := getCRForGroupKind("", "secret", userWithPassword.SecretRef, client.k8sDynClient)
	specData, _ = json.Marshal(secretCR.Object["data"])
	var k8sPw SecretResource
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

		// TODO - resolve the key ref and get the actual secret
		gcfg.CookieStoreKey = cfg.CookieStoreKeyRef

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
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					logger.Info("Watcher channel closed")
					return
				}
				logger.Infof("event received: %+v", event)
				eventHandler(event.Object)
			}
		}
	}()
}
