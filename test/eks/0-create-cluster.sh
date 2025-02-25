# https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html

aws eks create-cluster --region us-east-1 --name cluster-capitest --kubernetes-version 1.31 \
  --role-arn arn:aws:iam::728560144492:role/RoleEksOperator \
  --resources-vpc-config subnetIds=subnet-2c2d4870,subnet-b5294e9b

#,securityGroupIds=sg-a2573ae1

aws eks update-kubeconfig --region us-east-1 --name cluster-capitest

#   new kube config for arn:aws:eks:region-code:111122223333:cluster/my-cluster here:
#   /home/username/.kube/config

# https://percona.community/blog/2022/09/13/creating-a-kubernetes-cluster-on-amazon-eks-with-eksctl/
#https://eksctl.io/usage/minimum-iam-policies/

# Wehere does it save it?
aws configure --profile eksctl

eksctl create cluster -f cluster.yaml --profile eksctl


#eksctl create cluster --name cluster-capitest --region us-east-1 --version 1.31 --vpc-private-subnets subnet-2c2d4870,subnet-b5294e9b --without-nodegroup


johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ eksctl create cluster -f cluster.yaml --profile eksctl
2024-12-27 11:39:30 [ℹ]  eksctl version 0.199.0
2024-12-27 11:39:30 [ℹ]  using region us-east-1
2024-12-27 11:39:31 [ℹ]  setting availability zones to [us-east-1b us-east-1c]
2024-12-27 11:39:31 [ℹ]  subnets for us-east-1b - public:192.168.0.0/19 private:192.168.64.0/19
2024-12-27 11:39:31 [ℹ]  subnets for us-east-1c - public:192.168.32.0/19 private:192.168.96.0/19
2024-12-27 11:39:32 [ℹ]  nodegroup "ng-cass" will use "ami-00134b5b7db0c9684" [AmazonLinux2/1.30]
2024-12-27 11:39:32 [ℹ]  using EC2 key pair "eks-cluster-key"
2024-12-27 11:39:32 [ℹ]  using Kubernetes version 1.30
2024-12-27 11:39:32 [ℹ]  creating EKS cluster "cluster-capitest" in "us-east-1" region with un-managed nodes
2024-12-27 11:39:32 [ℹ]  1 nodegroup (ng-cass) was included (based on the include/exclude rules)
2024-12-27 11:39:32 [ℹ]  will create a CloudFormation stack for cluster itself and 1 nodegroup stack(s)
2024-12-27 11:39:32 [ℹ]  if you encounter any issues, check CloudFormation console or try 'eksctl utils describe-stacks --region=us-east-1 --cluster=cluster-capitest'
2024-12-27 11:39:32 [ℹ]  Kubernetes API endpoint access will use default of {publicAccess=true, privateAccess=false} for cluster "cluster-capitest" in "us-east-1"
2024-12-27 11:39:32 [ℹ]  CloudWatch logging will not be enabled for cluster "cluster-capitest" in "us-east-1"
2024-12-27 11:39:32 [ℹ]  you can enable it with 'eksctl utils update-cluster-logging --enable-types={SPECIFY-YOUR-LOG-TYPES-HERE (e.g. all)} --region=us-east-1 --cluster=cluster-capitest'
2024-12-27 11:39:32 [ℹ]  default addons vpc-cni, kube-proxy, coredns were not specified, will install them as EKS addons
2024-12-27 11:39:32 [ℹ]
2 sequential tasks: { create cluster control plane "cluster-capitest",
    2 sequential sub-tasks: {
        2 sequential sub-tasks: {
            1 task: { create addons },
            wait for control plane to become ready,
        },
        create nodegroup "ng-cass",
    }
}
2024-12-27 11:39:32 [ℹ]  building cluster stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:39:32 [ℹ]  deploying stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:40:02 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:40:33 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:41:33 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:42:34 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:43:34 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:44:35 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:45:35 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:46:36 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:47:36 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:48:37 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:49:37 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-cluster"
2024-12-27 11:49:39 [!]  recommended policies were found for "vpc-cni" addon, but since OIDC is disabled on the cluster, eksctl cannot configure the requested permissions; the recommended way to provide IAM permissions for "vpc-cni" addon is via pod identity associations; after addon creation is completed, add all recommended policies to the config file, under `addon.PodIdentityAssociations`, and run `eksctl update addon`
2024-12-27 11:49:39 [ℹ]  creating addon
2024-12-27 11:49:40 [ℹ]  successfully created addon
2024-12-27 11:49:40 [ℹ]  creating addon
2024-12-27 11:49:41 [ℹ]  successfully created addon
2024-12-27 11:49:41 [ℹ]  creating addon
2024-12-27 11:49:42 [ℹ]  successfully created addon
2024-12-27 11:51:43 [ℹ]  building nodegroup stack "eksctl-cluster-capitest-nodegroup-ng-cass"
2024-12-27 11:51:43 [ℹ]  --nodes-min=3 was set automatically for nodegroup ng-cass
2024-12-27 11:51:43 [ℹ]  --nodes-max=3 was set automatically for nodegroup ng-cass
2024-12-27 11:51:44 [ℹ]  deploying stack "eksctl-cluster-capitest-nodegroup-ng-cass"
2024-12-27 11:51:44 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-nodegroup-ng-cass"
2024-12-27 11:52:15 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-nodegroup-ng-cass"
2024-12-27 11:52:48 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-nodegroup-ng-cass"
2024-12-27 11:54:06 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-nodegroup-ng-cass"
2024-12-27 11:55:50 [ℹ]  waiting for CloudFormation stack "eksctl-cluster-capitest-nodegroup-ng-cass"
2024-12-27 11:55:50 [ℹ]  waiting for the control plane to become ready
2024-12-27 11:55:50 [✔]  saved kubeconfig as "/home/johndoe/.kube/config"
2024-12-27 11:55:50 [ℹ]  no tasks
2024-12-27 11:55:50 [✔]  all EKS cluster resources for "cluster-capitest" have been created
2024-12-27 11:55:51 [ℹ]  nodegroup "ng-cass" has 3 node(s)
2024-12-27 11:55:51 [ℹ]  node "ip-192-168-1-67.ec2.internal" is ready
2024-12-27 11:55:51 [ℹ]  node "ip-192-168-13-32.ec2.internal" is ready
2024-12-27 11:55:51 [ℹ]  node "ip-192-168-44-208.ec2.internal" is ready
2024-12-27 11:55:51 [ℹ]  waiting for at least 3 node(s) to become ready in "ng-cass"
2024-12-27 11:55:51 [ℹ]  nodegroup "ng-cass" has 3 node(s)
2024-12-27 11:55:51 [ℹ]  node "ip-192-168-1-67.ec2.internal" is ready
2024-12-27 11:55:51 [ℹ]  node "ip-192-168-13-32.ec2.internal" is ready
2024-12-27 11:55:51 [ℹ]  node "ip-192-168-44-208.ec2.internal" is ready
2024-12-27 11:55:51 [✔]  created 1 nodegroup(s) in cluster "cluster-capitest"
2024-12-27 11:55:52 [ℹ]  kubectl command should work with "/home/johndoe/.kube/config", try 'kubectl get nodes'
2024-12-27 11:55:52 [✔]  EKS cluster "cluster-capitest" in "us-east-1" region is ready
johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ kubectl pods
error: unknown command "pods" for "kubectl"

Did you mean this?
        logs
johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ kubectl get nodes
NAME                             STATUS   ROLES    AGE     VERSION
ip-192-168-1-67.ec2.internal     Ready    <none>   3m31s   v1.30.7-eks-59bf375
ip-192-168-13-32.ec2.internal    Ready    <none>   3m35s   v1.30.7-eks-59bf375
ip-192-168-44-208.ec2.internal   Ready    <none>   3m29s   v1.30.7-eks-59bf375
johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ kubectl get pods
No resources found in default namespace.
johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ kubectl get namespaces
NAME              STATUS   AGE
default           Active   11m
kube-node-lease   Active   11m
kube-public       Active   11m
kube-system       Active   11m
johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ kubectl get pods -nkube-public
No resources found in kube-public namespace.

johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ ssh -o StrictHostKeyChecking=no -i ~/.ssh/eks-cluster-key.pem UserEksOperator@34.205.172.27
UserEksOperator@34.205.172.27: Permission denied (publickey,gssapi-keyex,gssapi-with-mic).

johndoe@x15:/mnt/c/Users/John Doe/src/capillaries/test/eks$ eksctl delete cluster --name=cluster-capitest --profile eksctl
2024-12-27 12:08:28 [ℹ]  deleting EKS cluster "cluster-capitest"
2024-12-27 12:08:29 [ℹ]  will drain 1 unmanaged nodegroup(s) in cluster "cluster-capitest"
2024-12-27 12:08:29 [ℹ]  starting parallel draining, max in-flight of 1
2024-12-27 12:08:29 [ℹ]  cordon node "ip-192-168-1-67.ec2.internal"
2024-12-27 12:08:30 [ℹ]  cordon node "ip-192-168-13-32.ec2.internal"
2024-12-27 12:08:30 [ℹ]  cordon node "ip-192-168-44-208.ec2.internal"
2024-12-27 12:09:33 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:10:37 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:11:39 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:12:42 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:13:45 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:14:47 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:15:50 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:16:53 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:17:56 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:18:59 [!]  1 pods are unevictable from node ip-192-168-13-32.ec2.internal
2024-12-27 12:20:02 [ℹ]  cordon node "ip-192-168-25-134.ec2.internal"
2024-12-27 12:21:05 [!]  1 pods are unevictable from node ip-192-168-25-134.ec2.internal
2024-12-27 12:21:32 [ℹ]  cordon node "ip-192-168-54-109.ec2.internal"
2024-12-27 12:22:35 [!]  1 pods are unevictable from node ip-192-168-54-109.ec2.internal
^C
