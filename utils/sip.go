package utils

import (
	"os"
	"strconv"
)

// TODO call API to get proxy IPs
func GetSIPProxy() string {
	//return "proxy1";
	//return "52.60.126.237"
	//return "159.89.124.168"
	return os.Getenv("PROXY_HOST")
}

func GetSIPSecretKey() string {
	//return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
	//return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
	key := os.Getenv("LINEBLOCS_KEY")
	return key
}

func CreateSIPHeaders(domain, callerId, typeOfCall, apiCallId string, addedHeaders *[]string) map[string]string {
	headers := make(map[string]string)
	headers["SIPADDHEADER0"] = "X-LineBlocs-Key: " + GetSIPSecretKey()
	headers["SIPADDHEADER1"] = "X-LineBlocs-Domain: " + domain
	headers["SIPADDHEADER2"] = "X-LineBlocs-Route-Type: " + typeOfCall
	headers["SIPADDHEADER3"] = "X-LineBlocs-Caller: " + callerId
	headers["SIPADDHEADER4"] = "X-LineBlocs-API-CallId: " + apiCallId
	headerCounter := 5
	if addedHeaders != nil {
		for _, value := range *addedHeaders {
			headers["SIPADDHEADER"+strconv.Itoa(headerCounter)] = value
			headerCounter = headerCounter + 1
		}
	}
	return headers
}

func CreateSIPHeadersForSIPTrunkCall(domain, callerId, typeOfCall, apiCallId string, trunkAddr string) map[string]string {
	headers := make(map[string]string)
	headers["SIPADDHEADER0"] = "X-LineBlocs-Key: " + GetSIPSecretKey()
	headers["SIPADDHEADER1"] = "X-LineBlocs-Domain: " + domain
	headers["SIPADDHEADER2"] = "X-LineBlocs-Route-Type: " + typeOfCall
	headers["SIPADDHEADER3"] = "X-LineBlocs-Caller: " + callerId
	headers["SIPADDHEADER4"] = "X-LineBlocs-API-CallId: " + apiCallId
	headers["SIPADDHEADER5"] = "X-Lineblocs-User-SIP-Trunk-Addr: " + trunkAddr
	headers["SIPADDHEADER6"] = "X-Lineblocs-User-SIP-Trunk: true"
	return headers
}
