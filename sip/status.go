package sip

// SIP返回状态码信息
type StatusCodeItem struct {
	Code   int
	Reason string
}

// SIP返回状态码
var (
	// 1xx - 临时应答
	StatusTrying               = StatusCodeItem{100, "Trying"}
	StatusRinging              = StatusCodeItem{180, "Ringing"}
	StatusCallIsBeingForwarded = StatusCodeItem{181, "Call Is Being Forwarded"}
	StatusQueued               = StatusCodeItem{182, "Queued"}
	StatusSessionProgress      = StatusCodeItem{183, "Session Progress"}

	// 2xx - 成功处理
	StatusOK = StatusCodeItem{200, "OK"}

	// 3xx - 重定向
	StatusMultipleChoices    = StatusCodeItem{300, "Multiple Choices"}
	StatusMovedPermanently   = StatusCodeItem{301, "Moved Permanently"}
	StatusMovedTemporarily   = StatusCodeItem{302, "Moved Temporarily"}
	StatusUseProxy           = StatusCodeItem{305, "Use Proxy"}
	StatusAlternativeService = StatusCodeItem{380, "Alternative Service"}

	// 4xx - 客户端错误
	StatusBadRequest                  = StatusCodeItem{400, "Bad Request"}
	StatusUnauthorized                = StatusCodeItem{401, "Unauthorized"}
	StatusPaymentRequired             = StatusCodeItem{402, "Payment Required"}
	StatusForbidden                   = StatusCodeItem{403, "Forbidden"}
	StatusNotFound                    = StatusCodeItem{404, "Not Found"}
	StatusMethodNotAllowed            = StatusCodeItem{405, "Method Not Allowed"}
	StatusNotAcceptable               = StatusCodeItem{406, "Not Acceptable"}
	StatusProxyAuthenticationRequired = StatusCodeItem{407, "Proxy Authentication Required"}
	StatusRequestTimeout              = StatusCodeItem{408, "Request Timeout"}
	StatusGone                        = StatusCodeItem{410, "Gone"}
	StatusRequestEntityTooLarge       = StatusCodeItem{413, "Request Entity Too Large"}
	StatusRequestURITooLong           = StatusCodeItem{414, "Request-URI Too Long"}
	StatusUnsupportedMediaType        = StatusCodeItem{415, "Unsupported Media Type"}
	StatusUnsupportedURIScheme        = StatusCodeItem{416, "Unsupported URI Scheme"}
	StatusBadExtension                = StatusCodeItem{420, "Bad Extension"}
	StatusExtensionRequired           = StatusCodeItem{421, "Extension Required"}
	StatusIntervalTooBrief            = StatusCodeItem{423, "Interval Too Brief"}
	StatusNoResponse                  = StatusCodeItem{480, "No Response"}
	StatusCallTransactionDoesNotExist = StatusCodeItem{481, "Call/Transaction Does Not Exist"}
	StatusLoopDetected                = StatusCodeItem{482, "Loop Detected"}
	StatusTooManyHops                 = StatusCodeItem{483, "Too Many Hops"}
	StatusAddressIncomplete           = StatusCodeItem{484, "Address Incomplete"}
	StatusAmbigious                   = StatusCodeItem{485, "Ambiguous"}
	StatusBusyHere                    = StatusCodeItem{486, "Busy Here"}
	StatusRequestTerminated           = StatusCodeItem{487, "Request Terminated"}
	StatusNotAcceptableHere           = StatusCodeItem{488, "Not Acceptable Here"}
	StatusRequestPending              = StatusCodeItem{491, "Request Pending"}
	StatusUndecipherable              = StatusCodeItem{493, "Undecipherable"}

	// 5xx - 服务器错误
	StatusServerInternalError = StatusCodeItem{500, "Server Internal Error"}
	StatusNotImplemented      = StatusCodeItem{501, "Not Implemented"}
	StatusBadGateway          = StatusCodeItem{502, "Bad Gateway"}
	StatusServiceUnavailable  = StatusCodeItem{503, "Service Unavailable"}
	StatusServerTimeout       = StatusCodeItem{504, "Server Timeout"}
	StatusVersionNotSupported = StatusCodeItem{505, "Version Not Supported"}
	StatusMessageTooLarge     = StatusCodeItem{513, "Message Too Large"}

	// 6xx - 全局错误
	StatusBusyEverywhere       = StatusCodeItem{600, "Busy Everywhere"}
	StatusDecline              = StatusCodeItem{603, "Decline"}
	StatusDoesNotExistAnywhere = StatusCodeItem{604, "Does Not Exist Anywhere"}
	StatusUnacceptable         = StatusCodeItem{606, "Not Acceptable"}
)
