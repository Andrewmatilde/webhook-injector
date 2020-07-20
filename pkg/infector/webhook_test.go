package infector

import (
	"fmt"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"testing"
	"webhook-injector/pkg/config"
	"webhook-injector/pkg/reader"
)

func TestInject(t *testing.T) {

	//todo:
	cases := []struct {
		inputFile    string
		wantFile     string
		templateFile string
	}{
		{
			inputFile: "TestWebhookInject.yaml",
			wantFile:  "TestWebhookInject.patch",
		},
	}
	for _, c := range cases {
		input := filepath.Join("testdata/webhook", c.inputFile)
		//want := filepath.Join("testdata/webhook", c.wantFile)
		//templateFile := "TestWebhookInject_template.yaml"
		//if c.templateFile != "" {
		//	templateFile = c.templateFile
		//}
		config.InjectConfig = config.DefaultInjectConfig("../config/example/config.yaml")
		r := reader.FileReader{}
		podYAML, err := r.ReadOne(input)
		if err != nil {
			fmt.Printf("%v",err)
		}
		podJSON, err := yaml.YAMLToJSON(podYAML)
		got := inject(&v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: podJSON,
					},
				},
			})
		fmt.Printf("\n\n%v",string(got.Patch))
	}
}
