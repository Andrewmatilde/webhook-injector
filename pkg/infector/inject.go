package infector

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
	"webhook-injector/pkg/reader"

	"fmt"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
	"text/template"
)

type SidecarTemplateData struct {
	TypeMeta       *metav1.TypeMeta
	DeploymentMeta *metav1.ObjectMeta
	ObjectMeta     *metav1.ObjectMeta
	Spec           *corev1.PodSpec
	Values         map[string]interface{}
}

type SidecarInjectionSpec struct {
	// RewriteHTTPProbe indicates whether Kubernetes HTTP prober in the PodSpec
	// will be rewritten to be redirected by pilot agent.
	PodRedirectAnnot    map[string]string             `yaml:"podRedirectAnnot"`
	RewriteAppHTTPProbe bool                          `yaml:"rewriteAppHTTPProbe"`
	InitContainers      []corev1.Container            `yaml:"initContainers"`
	Containers          []corev1.Container            `yaml:"containers"`
	Volumes             []corev1.Volume               `yaml:"volumes"`
	DNSConfig           *corev1.PodDNSConfig          `yaml:"dnsConfig"`
	ImagePullSecrets    []corev1.LocalObjectReference `yaml:"imagePullSecrets"`
}

const (
	// ProxyContainerName is used by e2e integration tests for fetching logs
	ProxyContainerName = "istio-proxy"
)

func InjectionData(sidecarTemplate, version string, typeMetadata *metav1.TypeMeta, deploymentMetadata *metav1.ObjectMeta, spec *corev1.PodSpec,
	metadata *metav1.ObjectMeta) (
	*SidecarInjectionSpec, string, error) {

	// If DNSPolicy is not ClusterFirst, the Envoy sidecar may not able to connect to Istio Pilot.
	if spec.DNSPolicy != "" && spec.DNSPolicy != corev1.DNSClusterFirst {
		podName := potentialPodName(metadata)
		fmt.Printf("%q's DNSPolicy is not %q. The Envoy sidecar may not able to connect to Istio Pilot",
			metadata.Namespace+"/"+podName, corev1.DNSClusterFirst)
	}

	r := reader.FileReader{}
	//Todo: main path
	valuesConfig, err := r.ReadOne("./pkg/config/example/value.yaml")
	values := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(valuesConfig), &values); err != nil {
		fmt.Printf("Failed to parse values config: %v [%v]\n", err, valuesConfig)
		return nil, "", errors.WithMessage(err,"Could not parse configuration values;")
	}


	data := SidecarTemplateData{
		TypeMeta:       typeMetadata,
		DeploymentMeta: deploymentMetadata,
		ObjectMeta:     metadata,
		Spec:           spec,
		Values:         values,
	}

	funcMap := template.FuncMap{
		"formatDuration":      formatDuration,
		"isset":               isset,
		"excludeInboundPort":  excludeInboundPort,
		"includeInboundPorts": includeInboundPorts,
		"kubevirtInterfaces":  kubevirtInterfaces,
		"applicationPorts":    applicationPorts,
		"annotation":          getAnnotation,
		"valueOrDefault":      valueOrDefault,
		"toJSON":              toJSON,
		"toJson":              toJSON, // Used by, e.g. Istio 1.0.5 template sidecar-injector-configmap.yaml
		"fromJSON":            fromJSON,
		"structToJSON":        structToJSON,
		"protoToJSON":         protoToJSON,
		"toYaml":              toYaml,
		"indent":              indent,
		"directory":           directory,
		"contains":            flippedContains,
		"toLower":             strings.ToLower,
	}

	// Allows the template to use env variables from istiod.
	// Istiod will use a custom template, without 'values.yaml', and the pod will have
	// an optional 'vendor' configmap where additional settings can be defined.
	funcMap["env"] = func(key string, def string) string {
		val := os.Getenv(key)
		if val == "" {
			return def
		}
		return val
	}

	// Need to use FuncMap and SidecarTemplateData context
	funcMap["render"] = func(template string) string {
		bbuf, err := parseTemplate(template, funcMap, data)
		if err != nil {
			return ""
		}

		return bbuf.String()
	}

	bbuf, err := parseTemplate(sidecarTemplate, funcMap, data)
	if err != nil {
		return nil, "", err
	}

	var sic SidecarInjectionSpec
	if err := yaml.Unmarshal(bbuf.Bytes(), &sic); err != nil {
		// This usually means an invalid injector template; we can't check
		// the template itself because it is merely a string.
		fmt.Printf("Failed to unmarshal template %v\n %s", err, bbuf.String())
		return nil, "", errors.WithMessage(err,"failed parsing generated injected YAML (check Istio sidecar injector configuration):")
	}

	status := &SidecarInjectionStatus{Version: version}
	for _, c := range sic.InitContainers {
		status.InitContainers = append(status.InitContainers, c.Name)
	}
	for _, c := range sic.Containers {
		status.Containers = append(status.Containers, c.Name)
	}
	for _, c := range sic.Volumes {
		status.Volumes = append(status.Volumes, c.Name)
	}
	for _, c := range sic.ImagePullSecrets {
		status.ImagePullSecrets = append(status.ImagePullSecrets, c.Name)
	}
	statusAnnotationValue, err := json.Marshal(status)
	if err != nil {
		return nil, "", fmt.Errorf("error encoded injection status: %v", err)
	}
	return &sic, string(statusAnnotationValue), nil
}

func parseTemplate(tmplStr string, funcMap map[string]interface{}, data SidecarTemplateData) (bytes.Buffer, error) {
	var tmpl bytes.Buffer
	temp := template.New("inject")
	t, err := temp.Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		fmt.Printf("Failed to parse template: %v %v\n", err, tmplStr)
		return bytes.Buffer{}, err
	}
	if err := t.Execute(&tmpl, &data); err != nil {
		fmt.Printf("Invalid template: %v %v\n", err, tmplStr)
		return bytes.Buffer{}, err
	}

	return tmpl, nil
}

func formatDuration(in *types.Duration) string {
	dur, err := types.DurationFromProto(in)
	if err != nil {
		return "1s"
	}
	return dur.String()
}

func isset(m map[string]string, key string) bool {
	_, ok := m[key]
	return ok
}

func directory(filepath string) string {
	dir, _ := path.Split(filepath)
	return dir
}

func flippedContains(needle, haystack string) bool {
	return strings.Contains(haystack, needle)
}

func getPortsForContainer(container corev1.Container) []string {
	parts := make([]string, 0)
	for _, p := range container.Ports {
		if p.Protocol == corev1.ProtocolUDP || p.Protocol == corev1.ProtocolSCTP {
			continue
		}
		parts = append(parts, strconv.Itoa(int(p.ContainerPort)))
	}
	return parts
}

func getContainerPorts(containers []corev1.Container, shouldIncludePorts func(corev1.Container) bool) string {
	parts := make([]string, 0)
	for _, c := range containers {
		if shouldIncludePorts(c) {
			parts = append(parts, getPortsForContainer(c)...)
		}
	}

	return strings.Join(parts, ",")
}

// this function is no longer used by the template but kept around for backwards compatibility
func applicationPorts(containers []corev1.Container) string {
	return getContainerPorts(containers, func(c corev1.Container) bool {
		return c.Name != ProxyContainerName
	})
}

func includeInboundPorts(containers []corev1.Container) string {
	// Include the ports from all containers in the deployment.
	return getContainerPorts(containers, func(corev1.Container) bool { return true })
}

func kubevirtInterfaces(s string) string {
	return s
}

func structToJSON(v interface{}) string {
	if v == nil {
		return "{}"
	}

	ba, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("Unable to marshal %v", v)
		return "{}"
	}

	return string(ba)
}

func protoToJSON(v proto.Message) string {
	if v == nil {
		return "{}"
	}

	m := jsonpb.Marshaler{}
	ba, err := m.MarshalToString(v)
	if err != nil {
		fmt.Printf("Unable to marshal %v: %v", v, err)
		return "{}"
	}

	return ba
}


func toJSON(m map[string]string) string {
	if m == nil {
		return "{}"
	}

	ba, err := json.Marshal(m)
	if err != nil {
		fmt.Printf("Unable to marshal %v", m)
		return "{}"
	}

	return string(ba)
}

func fromJSON(j string) interface{} {
	var m interface{}
	err := json.Unmarshal([]byte(j), &m)
	if err != nil {
		fmt.Printf("Unable to unmarshal %s", j)
		return "{}"
	}

	fmt.Printf("%v", m)
	return m
}

func indent(spaces int, source string) string {
	res := strings.Split(source, "\n")
	for i, line := range res {
		if i > 0 {
			res[i] = fmt.Sprintf(fmt.Sprintf("%% %ds%%s", spaces), "", line)
		}
	}
	return strings.Join(res, "\n")
}

func toYaml(value interface{}) string {
	y, err := yaml.Marshal(value)
	if err != nil {
		fmt.Printf("Unable to marshal %v", value)
		return ""
	}

	return string(y)
}

func getAnnotation(meta metav1.ObjectMeta, name string, defaultValue interface{}) string {
	value, ok := meta.Annotations[name]
	if !ok {
		value = fmt.Sprint(defaultValue)
	}
	return value
}

func splitPorts(portsString string) []string {
	return strings.Split(portsString, ",")
}

func excludeInboundPort(port interface{}, excludedInboundPorts string) string {
	portStr := strings.TrimSpace(fmt.Sprint(port))
	if len(portStr) == 0 || portStr == "0" {
		// Nothing to do.
		return excludedInboundPorts
	}

	// Exclude the readiness port if not already excluded.
	ports := splitPorts(excludedInboundPorts)
	outPorts := make([]string, 0, len(ports))
	for _, port := range ports {
		if port == portStr {
			// The port is already excluded.
			return excludedInboundPorts
		}
		port = strings.TrimSpace(port)
		if len(port) > 0 {
			outPorts = append(outPorts, port)
		}
	}

	// The port was not already excluded - exclude it now.
	outPorts = append(outPorts, portStr)
	return strings.Join(outPorts, ",")
}

func valueOrDefault(value interface{}, defaultValue interface{}) interface{} {
	if value == "" || value == nil {
		return defaultValue
	}
	return value
}

// SidecarInjectionStatus contains basic information about the
// injected sidecar. This includes the names of added containers and
// volumes.
type SidecarInjectionStatus struct {
	Version          string   `json:"version"`
	InitContainers   []string `json:"initContainers"`
	Containers       []string `json:"containers"`
	Volumes          []string `json:"volumes"`
	ImagePullSecrets []string `json:"imagePullSecrets"`
}

// helper function to generate a template version identifier from a
// hash of the un-executed template contents.
func sidecarTemplateVersionHash(in string) string {
	hash := sha256.Sum256([]byte(in))
	return hex.EncodeToString(hash[:])
}

func potentialPodName(metadata *metav1.ObjectMeta) string {
	if metadata.Name != "" {
		return metadata.Name
	}
	if metadata.GenerateName != "" {
		return metadata.GenerateName + "***** (actual name not yet known)"
	}
	return ""
}


// overwriteClusterInfo updates cluster name and network from url path
// This is needed when webconfig config runs on a different cluster than webhook
func overwriteClusterInfo(containers []corev1.Container, path string) {
	res := strings.Split(path, "/")
	if len(res) >= 5 {
		// if len is less than 5, not enough length for /cluster/X/net/Y
		clusterName, clusterNetwork := "", ""
		clusterName = res[len(res)-3]
		clusterNetwork = res[len(res)-1]

		fmt.Printf("Updating cluster info based on clusterName: %s clusterNetwork: %s\n", clusterName, clusterNetwork)

		for i, c := range containers {
			if c.Name == ProxyContainerName {
				updateClusterInfo(&containers[i], clusterName, clusterNetwork)
			}
		}
	}

}

func updateClusterInfo(container *corev1.Container, clusterName, clusterNetwork string) {
	envVars := make([]corev1.EnvVar, 0)
	for _, env := range container.Env {
		if env.Name != "ISTIO_META_CLUSTER_ID" && env.Name != "ISTIO_META_NETWORK" {
			envVars = append(envVars, env)
		}
	}

	fmt.Printf("Appending env ISTIO_META_CLUSTER_ID: %s and ISTIO_META_NETWORK: %s\n", clusterName, clusterNetwork)
	envVars = append(envVars,
		corev1.EnvVar{Name: "ISTIO_META_CLUSTER_ID", Value: clusterName, ValueFrom: nil},
		corev1.EnvVar{Name: "ISTIO_META_NETWORK", Value: clusterNetwork, ValueFrom: nil})
	container.Env = envVars
}