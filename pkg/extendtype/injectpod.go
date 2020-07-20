package extendtype

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)


type PodItem struct {
	Pod *corev1.Pod
}

func (PodItem) TypeName() string {
	return "POD"
}

type InjectPod struct {
	PodItem
}

func (tp InjectPod) CheckAll(mustOkRuleList []TypeRule, oneOkRuleList []TypeRule) (bool,error) {
	ok, err := tp.CheckCNF(mustOkRuleList)
	if err != nil {
		return false,err
	}
	if !ok {
		return false,nil
	}
	ok, err = tp.CheckDNF(oneOkRuleList)
	if err != nil {
		return false,err
	}
	if !ok {
		return false,nil
	}
	return true,nil
}

func (tp InjectPod) CheckCNF(mustOkRuleList []TypeRule) (bool,error) {
	for _, tr := range mustOkRuleList {
		if !tr.Use {
			continue
		}
		ok,err := tr.Check(tp.PodItem)
		if err != nil {
			return false,errors.WithMessage(err, "Pod CNF Check failed;")
		}
		if !ok {
			return false,nil
		}
	}
	return true, nil
}

func (tp InjectPod) CheckDNF(oneOkRuleList []TypeRule) (bool,error) {
	for _, tr := range oneOkRuleList {
		if !tr.Use {
			continue
		}
		ok,err := tr.Check(tp.PodItem)
		if err != nil {
			return false,errors.WithMessage(err, "Pod DNF Check failed;")
		}
		if ok {
			return true,nil
		}
	}
	return false, nil
}
