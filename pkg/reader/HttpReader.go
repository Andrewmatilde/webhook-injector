package reader

import (
	"io/ioutil"
	"net/http"
	"sigs.k8s.io/yaml"
)


type HttpReader struct {
	Url   string
	Token string
}

func (inf HttpReader)Read() ([][]byte,error) {
	clt := http.Client{}
	req, err :=  http.NewRequest("GET", inf.Url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+ inf.Token)
	rsp, err := clt.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	type responseYaml struct {
		Items [] interface{} `yaml:"items"`
	}

	var  respy responseYaml
	if err := yaml.Unmarshal(body, &respy); err != nil {
		return nil,err
	}
	var items [][]byte

	for _, item := range respy.Items {
		var bt []byte
		if bt , err = yaml.Marshal(item); err != nil {
			return nil,err
		}
		items = append(items, bt)
	}
	return items,nil
}

