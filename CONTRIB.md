# Contribution guidelines

This is a begginer friendly project but it does require you to learn stuff.

## Knowledge Pre-requisites

Medium to good understanding of these topics below will help you get started quickly.

- OAuth
    - AuthZ code flow
    - JWT
- OpenID Connect (OIDC)
- Cookie based Sessions and Cookie handling
- RBAC
- Kubernetes basics

## How to run locally

We assume you have brew on Mac installed. If you are using Windows or *NIX, you need to look up online for the
equivalent ways to setup the env.

### Clone the project (one time)

```shell
git clone git@github.com:omrico/backbone.git
```

### Install Docker and Docker desktop (one time)

Visit https://docs.docker.com/get-docker/

### Install kubectl (the K8s command line tool) (one time)

```shell
brew install kubernetes-cli 
```

### Install Kind [(K8s in Docker)](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-with-a-package-manager) (one time)

```shell
brew install kind
```

### Start a kind server

```shell
kind create cluster
```

### Start the Backbone server (in cookie sessions mode)

```
KUBECONFIG=/path/to/.kube/config SYNC_INTERVAL_SECONDS=5 MODE=SESSIONS COOKIE_STORE_KEY=password go run cmd/main.go
```

### Create resources on the server

You will need to make sure your `KUBECONFIG` env var points to your kubeconfig file, usually under `~/.kube/config`.

Create password object:

```shell
kubectl apply -f resources/k8s-examples/SecretExample.yaml
```

Create user object:

```shell
kubectl apply -f resources/k8s-examples/UserExample.yaml
```

Create role object:

```shell
kubectl apply -f resources/k8s-examples/AdminRoleExample.yaml

```

Create role binding object, to attach a role to a user:

```shell
kubectl apply -f resources/k8s-examples/RoleBindingExample.yaml
```

### Test with HTTP client like Postman

Get userinfo:

```
GET /auth/sessions/userinfo

==> 403 Forbidden
{
  "errCode": "ERR.01.001",
  "errMessage": "user not logged in",
  "requestID": "fd80761d-23e0-408d-9ea7-1258c3e0100c"
}
```

Login:

```
POST /auth/sessions/login
Body:
{
    "username" : "omri@gopher.com",
    "password" : "ca$hc0w"
}

==> 200 OK
cookie set "backbone_session..."
```

Try Get userinfo again:

```
GET /auth/sessions/userinfo

==> 200 OK
{
    "username": "omri@gopher.com
    "roles": {
        "admin": [
            "ALL"
        ]
    }
}
```


