# checkup-launcher
## Build Instructions
1. Build the image:
```bash
$ cd checkup-launcher
$ ./build-image
```

Note: you can use other container engine to build the image, for example:
```bash
$ cd checkup-launcher
$ CRI=docker ./build-image
```
2. The result shall be an image tagged: `checkup-launcher:latest`

## Deployment Instructions
1. Push the built image to a registry of your choice.
2. See example manifest under `checkups/echo/manifests/echo-checkup-framework-job.yaml`.