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

	digest, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return digest[len(digest)-60 : len(digest)-1], nil
}
