module github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency

require kubevirt.io/client-go v0.49.0

replace github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47

require k8s.io/apimachinery v0.20.2

replace (
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
)

require (
	github.com/google/goexpect v0.0.0-20210430020637-ab937bf7fd6f
	google.golang.org/grpc v1.31.0
	k8s.io/api v0.23.1
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	kubevirt.io/api v0.0.0-20220111180619-bd15f69822b9
	sigs.k8s.io/yaml v1.2.0
)

go 1.16
