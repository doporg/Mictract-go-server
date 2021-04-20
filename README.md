# mictract

Mictract (MICro-serviced smart conTRACT), focus on applying traditional DevOps tools to the smart contract development process, based on [Hyperledger / Fabric](https://github.com/hyperledger/fabric/).

WIP now.



## Deployment



### NFS & DB

1. install `nfs-utils`

2. expose the data directory, which will be used for storing networks. Note: this path is corresponds to `config.NFS_EXPOSED_PATH` in the source code.
   ```
   /var/mictract    *(rw,all_squash,fsid=0,anonuid=0,anongid=0,insecure)
   ```

3. run it.
   ```
   systemctl enable --now nfs-server
   ```

4. configure database:
   1. start your mysql database.
      For example, use [mysql](https://hub.docker.com/_/mysql) on docker:
      ```
      docker run --name mic-mysql -e MYSQL_ROOT_PASSWORD=123456 -p 3306:3306 -d mysql:8
      ```

   2. expose external database service into k8s.
      Edit `k8s/mysql-svc.yaml` and `k apply -f k8s/mysql-svc.yaml`.

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

### Prometheus [optional]

We use [Prometheus]() for network monitoring.

1. Add prometheus to your k8s. (If you have one, you should ensure your prometheus can monitor k8s cAdvisor)
   ```
   kubectl apply -k k8s/prometheus
   ```

2. Check if prometheus is running.
   ```
   kubectl get pod
   kubectl get svc
   ```
   If it's correct, you can see the `prometheus` pod and service running.
   ```
   NAME                                              READY   STATUS             RESTARTS   AGE
   ...
   prometheus-7dddd47bcf-7w8td                       1/1     Running            0          2s


   NAME                             TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                                      AGE
   ...
   prometheus                       NodePort       10.109.131.55    <none>        9090:30500/TCP                               2s
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
5. configure mysql and nfs
   ```shell
   export NFS_SERVER_URL=x.x.x.x
   export DB_SERVER_URL=x.x.x.x
   export DB_PW=YourDatabasePassword
   ```

6. run dlv when you need to *debug*.
   ```shell
   dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient
   
   // Or you just want simplest unit test on a function:
   cd src/test
   go test * -test.run TestListNetworks -v
   ```
   Or, when you need to *run* it, `go run main.go` via ssh terminal directly. 

