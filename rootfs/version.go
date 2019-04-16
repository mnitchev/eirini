package rootfs

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type Digester struct {
	BitsURL string
}

func (a *Digester) Get() ([]byte, error) {
	endpoint := fmt.Sprintf("%s/v2/eirinifs", a.BitsURL)
	resp, err := http.Get(endpoint)
	if err != nil {
		return []byte{}, err
	}

	return ioutil.ReadAll(resp.Body)
}
