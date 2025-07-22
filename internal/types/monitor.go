package types

type HttpConfig struct {
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	ExpectedStatus int               `json:"expected_status"`
	Timeout        int               `json:"timeout"`
}

type DNSConfig struct {
	Domain     string `json:"domain"`
	RecordType string `json:"record_type"` // A, AAAA, CNAME, MX, TXT, etc.
	Expected   string `json:"expected"`    // Expected IP/value (optional)
	Timeout    int    `json:"timeout"`     // Timeout in seconds
}

