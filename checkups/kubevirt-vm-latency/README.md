# Kubevirt Network Latency Checkup

This checkup performs network latency measurement between two
Kubevirt VM's.

## Build Instructions
```bash
# build Kubevirt network latency image
$ ./build/build-image

#
$ checkups/kubevirt-vm-latency/build/build-image

# override CRI to use a different container runtime
$ CRI=docker ./build/build-image
```
# Notes
If you use Goland IDE you might need to enable go modules integration
in order for the IDE to recognize all dependencies:
Settings > Go > Go Modules > Enable Go module integration