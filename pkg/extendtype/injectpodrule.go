package extendtype

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"log"
	"webhook-injector/pkg/config"
)

var MustOkpodTypeRuleList = []TypeRule{
	{Rule:podValid{}, Use: false},
	{Rule:podInjectValid{}, Use: true},
	{Rule:podPolicyValid{}, Use: true},
	{Rule:podInNeverInjectSelector{}, Use: true},
}

var OneOkpodTypeRuleList = []TypeRule{
	{Rule:podInAlwaysInjectSelector{}, Use: true},
	{Rule:podAnnotationOk{}, Use: true},
	{Rule:podPolicyEnabled{}, Use: true},
}

type podValid struct {}

func (podValid)Check(item Item) (bool,error) {
	if item.TypeName() != "POD" {
		return false,errors.Wrap(errors.New("Type Error: Not Pod;"),
			"podValid check failed;")
	}
	pod := item.(PodItem).Pod
	if len(pod.Spec.Containers) < 1{
		return false,errors.Wrap(errors.New("Pod Error: Pod Container not exist;"),
			"podValid check failed;")
	}
	return true,nil
}

type podInjectValid struct {}

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

func (podInjectValid) Check(item Item) (bool, error) {
	pod := item.(PodItem).Pod
	if pod.Spec.HostNetwork {
		return false,nil
	}
	for _, namespace := range ignoredNamespaces {
		if pod.ObjectMeta.Namespace == namespace {
			return false,nil
		}
	}
	annos := pod.ObjectMeta.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	// ToDo: write annos code

	return true,nil
}

type podPolicyValid struct {}

func (podPolicyValid) Check(item Item) (bool, error) {
	// Todo: write config change struct
	if config.InjectConfig.Policy != config.InjectionPolicyDisabled &&
		config.InjectConfig.Policy != config.InjectionPolicyEnabled {
		return false, errors.Errorf("Illegal value for autoInject:%s, must be one of [%s,%s]. Auto injection disabled!",
			config.InjectConfig.Policy, config.InjectionPolicyDisabled, config.InjectionPolicyEnabled)
	}
	return true,nil
}

type podInNeverInjectSelector struct {}

func (podInNeverInjectSelector)Check(item Item) (bool,error) {
	pod := item.(PodItem).Pod
	for _, neverSelector := range config.InjectConfig.NeverInjectSelector {
		selector, err := metav1.LabelSelectorAsSelector(&neverSelector)
		if err != nil {
			return false, errors.Errorf("Invalid selector for NeverInjectSelector: %v (%v)",
				neverSelector, err)
		} else if !selector.Empty() && selector.Matches(labels.Set(pod.ObjectMeta.Labels)) {
			//Todo: new log
			log.Printf("Explicitly disabling injection for Pod %s due to Pod labels matching NeverInjectSelector config map entry.",
				pod.ObjectMeta.Namespace)
			return false,nil
		}
	}
	return true,nil
}

type podAnnotationOk struct {}

func (podAnnotationOk)Check(item Item) (bool,error) {
	//todo:
	return false,nil
}

type podInAlwaysInjectSelector struct {}

func (podInAlwaysInjectSelector)Check(item Item) (bool,error) {
	pod := item.(PodItem).Pod
	for _, alwaysSelector := range config.InjectConfig.AlwaysInjectSelector {
		selector, err := metav1.LabelSelectorAsSelector(&alwaysSelector)
		if err != nil {
			return false, errors.Errorf("Invalid selector for AlwaysInjectSelector: %v (%v)", alwaysSelector, err)
		} else if !selector.Empty() && selector.Matches(labels.Set(pod.ObjectMeta.Labels)) {
			//Todo: new log
			log.Printf("Explicitly enabling injection for Pod %s due to Pod labels matching AlwaysInjectSelector config map entry.",
				pod.ObjectMeta.Namespace)
			return true,nil
		}
	}
	return false,nil
}

type podPolicyEnabled struct {}

func (podPolicyEnabled)Check(item Item) (bool,error) {
	if config.InjectConfig.Policy == config.InjectionPolicyEnabled {
		return true, nil
	}
	return false,nil
}
