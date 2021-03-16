# mictract

Mictract (MICro-serviced smart conTRACT), focus on applying traditional DevOps tools to the smart contract development process, based on [Hyperledger / Fabric](https://github.com/hyperledger/fabric/).

WIP now.



## Deployment



### NFS

1. install `nfs-utils`

2. expose the data directory, which will be used for storing networks. Note: this path is corresponds to `config.NFS_EXPOSED_PATH` in the source code.
   ```
   /var/mictract    *(rw,all_squash,fsid=0,anonuid=0,anongid=0,insecure)
   ```

3. run it.
   ```
   systemctl enable --now nfs-server
   ```

4. add hosts for NFS server, or edit `config.NFS_SERVER_URL`.

   ```
   x.x.x.x             nfs-server
   ```

5. put your k8s config and `template/configtx.yaml.tpl` and `template/channel.yaml.tpl` and `scripts` on `config.NFS_EXPOSED_PATH`, your `kube-config.yaml` should be like this:
   
   Note: the `certificate-authority-data` and `client-key-data` are base64 encoded.
   
   ```yaml
   apiVersion: v1
   clusters:
   - cluster:
     certificate-authority-data: LS0tL...UZJQ0FURS0tLS0tCg==
     extensions:
     server: https://kubernetes:443
     name: minikube
     contexts:
   - context:
     cluster: minikube
     namespace: default
     user: minikube
     name: minikube
     current-context: minikube
     kind: Config
     preferences: {}
     users:
   - name: minikube
     user:
     client-certificate-data: LS0tL...CBDRVJUSUZJQ0FURS0tLS0tCg==
     client-key-data: LS0tL...FJTQSBQUklWQVRFIEtFWS0tLS0tCg==
   ```
   
6. check your network directory, which should be like this:
   ```text
   mictract
   ├── configtx.yaml.tpl
   ├── channel.yaml.tpl
   ├── kube-config.yaml
   └── scripts
   ```

### Development environment [optional] 

1. setup minikube.
   ```shell
   curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
   sudo install minikube-linux-amd64 /usr/local/bin/minikube

   minikube start
   ```
   Then you can use `minikube ip` as kubernetes api-server ip.

2. edit `dev-deploy.yaml`, and run mictract-dev env, where the dlv and sftp is deployed.
   ```shell
   k apply -k k8s/dev
   ```

3. use sftp to sync your code.

4. [optional] change shell.
   ```shell
   chsh -s /bin/zsh
   ```

4. run dlv when you need to *debug*.
   ```shell
   dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient
   
   // Or you just want simplest unit test on a function:
   cd src/test
   go test * -test.run TestListNetworks -v
   ```
   Or, when you need to *run* it, `go run main.go` via ssh terminal directly. 

