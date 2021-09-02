# New Visitors Site Operator

This Operator adds some functions based on Visitors Site Operator in https://developers.redhat.com/books/kubernetes-operators to achieve all 5 levels of Operator capability model.

## Level 1: basic install

For installation, enter the main directory and run: 

```shell
make deploy
```

or 

```shell
make install run
```

The necessary Custom Resource Definition called VisitorsApp should be automatically created. Then create a Custom Resource (CR) by applying the yaml file in the config/samples/ folder:

```shell
kubectl apply -f config/samples/example.com_v1beta1_visitorsapp.yaml
```

A CR called visitorsapp-sample should have been generated. But note that by now, neither frontend pods nor backend pods are created. This is because they are all waiting for the database pods to be up. However, our operator no longer create MySQL pods by ourselves like the old version does. In order to make it easier to achieve capability levels 3 to 5, an open source MySQL operator is needed to create a MySQL cluster. And in our demo, we choose presslabs (or bitpoke) MySQL Operator.

Helm, a package manager for Kubernetes can be helpful for an easy installation of the Operators published on artifacthub.io. Only one single command should be enough. Make sure that you have Helm installed on your computer, and have added presslabs to your repositories:

```shell
helm install mysql-operator presslabs/mysql-operator
```

Then, you are able to create the MySQL cluster by first creating the database’s secret, and then the database cluster itself:

```shell
kubectl apply -f config/samples/mysql/example-cluster-secret.yaml
kubectl apply -f config/samples/mysql/example-cluster.yaml
```

Now, after some time, you should see that pods of database, backend, and frontend are all running as expected. You can test your application is running by open your browser and go to the site: http://<minikubeIP>:30686/. Now each page refresh can add another visit record to the table displayed. 

30686 is the default frontend service node port, which can be set in your VisitorsApp CR yaml file. And you can get your minikube IP by running the minikube command: 

```shell
minikube ip
```

To uninstall the application, just delete the CR:

```shell
kubectl delete -f config/samples/example.com_v1beta1_visitorsapp.yaml
```

## Level 2: seamless upgrades

Upgrading the application is simple. Modify the file content in config/samples/example.com_v1beta1_visitorsapp.yaml or config/samples/mysql/example-cluster.yaml, and use kubectl apply to apply those changes. Variables like homepage’s title, pod replicas and MySQL version can all be changed and applied to the application.


## Level 3: full lifecycle 

Functionalities including backup and restore are within the capabilities of presslabs MySQL operator. A remote platform for data storage like AWS or Google Cloud Service is required. 

First, enter the backup credentials in example-backup-secret.yaml, example-backup.yaml, and example-cluster.yaml in config/samples/mysql/ folder. Each time you want store the visitors’ records to remote, apply the example-backup.yaml file. A URL should indicate the path to your remote storage. And each time your want to restore the information, specify the initBucketURL in example-cluster.yaml to be the remote storage path and apply the yaml file.


## Level 4: deep insights

In this level, functionalities including monitoring and alerting can be realized with the help of prometheus, an open-source toolkit for monitoring the state of cluster, and sending alert when any rule is broken. For installation, we can use Helm again. After adding prometheus-community to your repositories, run:

```shell
helm install prometheus prometheus-community/kube-prometheus-stack
```

In order for prometheus to understand the metrics sent by MySQL, a MySQL exporter is need to change the metrics to the proper form that can be understood by prometheus:

```shell
helm install mysql-exporter prometheus-community/prometheus-mysql-exporter -f config/samples/mysql-exporter/values.yaml
```

After doing port-forward for the prometheus service, you can go to the prometheus page (localhost:9090) to check your cluster components that are being monitored by prometheus as well as all the rules and alerts. A service monitor for MySQL should be one of the targets if MySQL exporter is correctly installed.

```shell
kubectl port-forward service/prometheus-operated 9090
```

## Level 5: auto pilot

Kubernetes has an API resource called Horizontal Pod Autoscaler (HPA) that is able to auto-scale the number of pod replicas. We are going to make use of this technique to achieve auto-scaling of the database pods.

Before creating an HPA, prometheus adapter must be installed to make HPA able to collect the metrics from prometheus:

```shell
helm install prometheus-adapter prometheus-community/prometheus-adapter -f config/samples/prometheus-adapter/values.yaml
```

### Backend Auto-scaling

To enable the auto-scaling of backend, we first change backendAutoScaling in the spec of config/samples/example.com_v1beta1_visitorsapp.yaml to "true", and apply it. Then apply the HPA for backend.

```shell
kubectl apply -f config/samples/example.com_v1beta1_visitorsapp.yaml
kubectl apply -f config/samples/autoscaling/backend-hpa.yaml
```

### Database Auto-scaling

Auto-scaling for database is a little bit more tricky. The CRD for MysqlCluster should be modified in order to let HPA realize which pods belong to the cluster, so that auto-scaling for the MysqlCluster can become possible:

```shell
kubectl apply -f config/samples/autoscaling/mysql.presslabs.org_mysqlclusters.yaml
```

Make sure that you restart the cluster before continuing. 
Then, apply the HPA to auto-scale the database cluster based on the CPU utilization of the MySQL pods. Of course other criteria can be used to replace this one, as long as their information is accessible from prometheus.

```shell
kubectl apply -f config/samples/autoscaling/database-hpa.yaml
```