# Platform AWS Controller
This controller wraps the hypershift CLI, and builds out the required AWS Infrastructure, when the HoestedCluster resource is created.

## Setup
1. Follow the steps to install hypershift on an OpenShift cluster, see main README.md
2. Run the controller (uses the CLI connection to the Hypershift Management Cluster)
```
go run platform/main.go
```
3. Create a secret with the AWS key credentials
    ```
    ---
    apiVersion: v1
    stringData:
    aws_access_key_id: MY_AWS_KEY_ID
    aws_secret_access_key: MY_AWS_SECRET_KEY
    kind: Secret
    metadata:
    name: aws               # DO NOT CHANGE
    namespace: clusters     # DO NOT CHANGE
    type: Opaque
    ```
4. Create the OpenShift pull-secret
    ```
    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: pull-secret    # Same name used in the HostedCluster resource Spec
      namespace: clusters
    stringData:
      .dockerconfigjson: |-
        MY_OPENSHIFT_PULL_SECRET_GOES_HERE
    ```
5. Apply a HostedCluster resource
   ```
   # Change the baseDomain value in the YAML file before applying it to the ManagementCluster
   oc apply -f ./demo01.yaml
   ```
6. You will see the Platform AWS configuration log messaging in the CLI where you ran the `go run` command
7. Delete the cluster by deleting the HostedCluster resource.
    ```
    oc delete -f ./demo01.yaml
    ```

A NodePool is also created as part of the work initiated by created the HostedCluster.  This this connection will be cut in the next iteration of the prototype.
