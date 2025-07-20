package types

type HttpConfig struct {
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	ExpectedStatus int               `json:"expected_status"`
	Timeout        int               `json:"timeout"`
}
