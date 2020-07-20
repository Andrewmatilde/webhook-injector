package extendtype

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"webhook-injector/pkg/config"
)

func TestInjectPod(t *testing.T) {
	config.InjectConfig = config.DefaultInjectConfig("./config/example/config.yaml")
	podSpec := &corev1.PodSpec{}
	//Todo: more test
	cases := []struct {
		policy config.InjectionPolicy
		pod    *corev1.Pod

		want    bool
	}{
		{
			pod: &corev1.Pod{
				Spec: *podSpec,
				ObjectMeta: metav1.ObjectMeta{
					Name:        "no-policy",
					Namespace:   "test-namespace",
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
		{
			policy: "wrong-police",
			pod: &corev1.Pod{
				Spec: *podSpec,
				ObjectMeta: metav1.ObjectMeta{
					Name:        "wrong-policy",
					Namespace:   "test-namespace",
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
		{
			policy: config.InjectionPolicyEnabled,
			pod: &corev1.Pod{
				Spec: *podSpec,
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-policy",
					Namespace: "test-namespace",
				},
			},
			want: true,
		},
		{
			//todo:
			policy: config.InjectionPolicyEnabled,
			pod: &corev1.Pod{
				Spec: *podSpec,
				ObjectMeta: metav1.ObjectMeta{
					Name:        "force-on-policy",
					Namespace:   "test-namespace",
					Annotations: map[string]string{"name": "true"},
				},
			},
			want: true,
		},
		{
			policy: config.InjectionPolicyDisabled,
			pod: &corev1.Pod{
				Spec: *podSpec,
				ObjectMeta: metav1.ObjectMeta{
					Name:        "disable-policy",
					Namespace:   "test-namespace",
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
	}

	for _, c := range cases {
		ip := InjectPod{PodItem{Pod:c.pod}}
		MustOkpodTypeRuleList[0].Use = false
		config.InjectConfig.Policy = c.policy
		if got,_ := ip.CheckAll(MustOkpodTypeRuleList,OneOkpodTypeRuleList); got != c.want {
			t.Errorf("ip.CheckAll(%v) got %v want %v",ip.Pod.ObjectMeta.Name,got, c.want)
		}
	}
}
