# Echo checkup
This is an example checkup, used as a reference for creating more realistic checkups for the K8s checkup framework.

The checkup receives an environment variable named "MESSAGE", and writes it to a provided ConfigMap object.
## Build Instructions
### Prerequisites
- You have [podman](https://podman.io/) or other container engine capable of building images.
### Steps
1. Change directory to the echo checkup's root directory:
`cd checkups/echo`
2. Run the `build-image` script:
`./build-image`
3. You can provide other CRI using:
`CRI=docker ./build-image`

## Manual Execution Instructions
### Prerequisites
- The checkup container is built, tagged and stored in a registry accessible to your cluster.
- You have "Admin" permissions on the K8s cluster.
- kubectl is configured to connect to your cluster.
### Steps
1. Deploy the checkup required "environment" using:
`kubectl create -f manifests/echo-checkup-env.yaml`
2. Edit the image tag in `manifests/echo-checkup-job.yaml`.
3. You can potentially edit the checkup MESSAGE environment variable as well.
4. Run the checkup Job using:
`kubectl create -f manifests/echo-checkup-job.yaml`
5. To get the checkup results:
`kubectl get configmap echo-checkup-result -n echo-checkup-ns -o yaml > results.yaml`
6. To remove the created environment:
`kubectl delete -f manifests/echo-checkup-job.yaml`
