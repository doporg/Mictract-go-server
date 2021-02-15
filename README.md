# mictract

Mictract (MICro-serviced smart conTRACT), focus on applying traditional DevOps tools to the smart contract development process, based on [Hyperledger / Fabric](https://github.com/hyperledger/fabric/).

WIP now.

## Deployment

### Basic env (NFS)

...

### Development env

1. setup minikube.
  ```shell
  curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
  sudo install minikube-linux-amd64 /usr/local/bin/minikube

  minikube start
  ```

2. edit `dev-deploy.yaml`, and run mictract-dev env, where the dlv and sftp is deployed.
  ```shell
  k apply -k k8s/dev
  ```

3. use sftp to sync your code.

4. [optional] change shell.
  ```shell
  chsh -s /bin/zsh
  ```

4. run dlv when you need to debug.
  ```shell
  dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient
  ```
  Or, when you need to run it, `go run src/main.go` via ssh terminal directly.

