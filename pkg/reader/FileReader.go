package reader

import "io/ioutil"

type FileReader struct {
	PathSet []string
}

func (f FileReader)Read() ([][]byte ,error) {
	var items [][]byte

	for _, path := range f.PathSet {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		items = append(items, data)
	}
	return items,nil
}

func (f FileReader)ReadOne(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data,nil
}

