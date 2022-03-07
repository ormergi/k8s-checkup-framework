# k8s-checkup-framework

# Checkup's Launcher
## Build Instructions
```bash
# build checkup-framework's image
$ ./build/build-image

# override CRI to use a different container runtime
$ CRI=docker ./build/build-image
```

## Deployment Instructions
1. Push the built image to a registry of your choice.
2. See example manifest under:
`checkups/echo/manifests/echo-checkup-framework-job.yaml`.