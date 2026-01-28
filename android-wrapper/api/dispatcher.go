package api

import (
	"mihomo_android_wrapper/contract"
)

type DispatchResult struct {
	Response  contract.Response
	AfterSend func()
}

type Dispatcher struct {
	Service contract.Service
}

// New creates a Dispatcher that routes contract.Action to Service.
func New(service contract.Service) *Dispatcher {
	return &Dispatcher{Service: service}
}

// Dispatch routes an Action to Service and builds a response.
// AfterSend is used for side effects that must happen after the response is sent (for example, crash tests).
func (d *Dispatcher) Dispatch(action contract.Action) DispatchResult {
	result := DispatchResult{
		Response: contract.Response{
			ID:     action.ID,
			Method: action.Method,
			Code:   0,
		},
	}

	fail := func(data any) DispatchResult {
		result.Response.Code = -1
		result.Response.Data = data
		return result
	}

	success := func(data any) DispatchResult {
		result.Response.Code = 0
		result.Response.Data = data
		return result
	}

	switch action.Method {
	case contract.InitClashMethod:
		var params contract.InitParams
		if err := decodeJSON(action.Data, &params); err != nil {
			return fail("initClash: invalid params: " + err.Error())
		}
		return success(d.Service.InitClash(params))
	case contract.GetVersionMethod:
		return success(d.Service.GetVersion())
	case contract.GetIsInitMethod:
		return success(d.Service.GetIsInit())
	case contract.ForceGcMethod:
		d.Service.ForceGC()
		return success(true)
	case contract.ShutdownMethod:
		return success(d.Service.Shutdown())
	case contract.ValidateConfigMethod:
		path, err := decodeString(action.Data)
		if err != nil {
			return fail("validateConfig: invalid params: " + err.Error())
		}
		return success(d.Service.ValidateConfig(path))
	case contract.GetConfigMethod:
		path, err := decodeString(action.Data)
		if err != nil {
			return fail("getConfig: invalid params: " + err.Error())
		}
		cfg, err := d.Service.GetConfig(path)
		if err != nil {
			return fail(err.Error())
		}
		return success(cfg)
	case contract.UpdateConfigMethod:
		payload, err := decodeString(action.Data)
		if err != nil {
			return fail("updateConfig: invalid params: " + err.Error())
		}
		return success(d.Service.UpdateConfig(payload))
	case contract.SetupConfigMethod:
		payload, err := decodeString(action.Data)
		if err != nil {
			return fail("setupConfig: invalid params: " + err.Error())
		}
		return success(d.Service.SetupConfig(payload))
	case contract.GetProxiesMethod:
		return success(d.Service.GetProxies())
	case contract.ChangeProxyMethod:
		var params contract.ChangeProxyParams
		if err := decodeJSON(action.Data, &params); err != nil {
			return fail("changeProxy: invalid params: " + err.Error())
		}
		return success(d.Service.ChangeProxy(params))
	case contract.GetTrafficMethod:
		onlyProxy, err := decodeBool(action.Data)
		if err != nil {
			return fail("getTraffic: invalid params: " + err.Error())
		}
		return success(d.Service.GetTraffic(onlyProxy))
	case contract.GetTotalTrafficMethod:
		onlyProxy, err := decodeBool(action.Data)
		if err != nil {
			return fail("getTotalTraffic: invalid params: " + err.Error())
		}
		return success(d.Service.GetTotalTraffic(onlyProxy))
	case contract.ResetTrafficMethod:
		d.Service.ResetTraffic()
		return success(true)
	case contract.AsyncTestDelayMethod:
		data, err := decodeString(action.Data)
		if err != nil {
			return fail("asyncTestDelay: invalid params: " + err.Error())
		}
		return success(d.Service.AsyncTestDelay(data))
	case contract.GetConnectionsMethod:
		return success(d.Service.GetConnections())
	case contract.CloseConnectionsMethod:
		return success(d.Service.CloseConnections())
	case contract.ResetConnectionsMethod:
		return success(d.Service.ResetConnections())
	case contract.CloseConnectionMethod:
		id, err := decodeString(action.Data)
		if err != nil {
			return fail("closeConnection: invalid params: " + err.Error())
		}
		return success(d.Service.CloseConnection(id))
	case contract.GetExternalProvidersMethod:
		return success(d.Service.GetExternalProviders())
	case contract.GetExternalProviderMethod:
		name, err := decodeString(action.Data)
		if err != nil {
			return fail("getExternalProvider: invalid params: " + err.Error())
		}
		return success(d.Service.GetExternalProvider(name))
	case contract.UpdateGeoDataMethod:
		payload, err := decodeString(action.Data)
		if err != nil {
			return fail("updateGeoData: invalid params: " + err.Error())
		}
		return success(d.Service.UpdateGeoData(payload))
	case contract.SideLoadExternalProviderMethod:
		payload, err := decodeString(action.Data)
		if err != nil {
			return fail("sideLoadExternalProvider: invalid params: " + err.Error())
		}
		return success(d.Service.SideLoadExternalProvider(payload))
	case contract.UpdateExternalProviderMethod:
		name, err := decodeString(action.Data)
		if err != nil {
			return fail("updateExternalProvider: invalid params: " + err.Error())
		}
		return success(d.Service.UpdateExternalProvider(name))
	case contract.GetCountryCodeMethod:
		ip, err := decodeString(action.Data)
		if err != nil {
			return fail("getCountryCode: invalid params: " + err.Error())
		}
		return success(d.Service.GetCountryCode(ip))
	case contract.GetMemoryMethod:
		return success(d.Service.GetMemory())
	case contract.StartLogMethod:
		d.Service.StartLog()
		return success(true)
	case contract.StopLogMethod:
		d.Service.StopLog()
		return success(true)
	case contract.StartListenerMethod:
		return success(d.Service.StartListener())
	case contract.StopListenerMethod:
		return success(d.Service.StopListener())
	case contract.UpdateDnsMethod:
		data, err := decodeString(action.Data)
		if err != nil {
			return fail("updateDns: invalid params: " + err.Error())
		}
		d.Service.UpdateDns(data)
		return success(true)
	case contract.CrashMethod:
		result.AfterSend = d.Service.Crash
		return success(true)
	case contract.DeleteFileMethod:
		path, err := decodeString(action.Data)
		if err != nil {
			return fail("deleteFile: invalid params: " + err.Error())
		}
		return success(d.Service.DeleteFile(path))
	default:
		return fail("unknown method")
	}
}
