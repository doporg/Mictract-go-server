package test

import (
	"bytes"
	"mictract/router"
	"encoding/json"
	"io"
	"net/http/httptest"

	_ "mictract/init"
)

var R = router.GetRouter()

func Base(method string, uri string, body io.Reader) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, uri, body)
	w := httptest.NewRecorder()
	R.ServeHTTP(w, r)
	return w
}

func Parse(param interface{}) io.Reader {
	jsonByte, _ := json.Marshal(param)
	return bytes.NewReader(jsonByte)
}

func Get(uri string, reader io.Reader) *httptest.ResponseRecorder {
	return Base("GET", uri, reader)
}

func Delete(uri string) *httptest.ResponseRecorder {
	return Base("DELETE", uri, nil)
}

func Post(uri string, reader io.Reader) *httptest.ResponseRecorder {
	return Base("POST", uri, reader)
}
