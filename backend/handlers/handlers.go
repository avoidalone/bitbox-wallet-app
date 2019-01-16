// Copyright 2018 Shift Devices AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime/debug"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/digitalbitbox/bitbox-wallet-app/backend"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/accounts"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc"
	accountHandlers "github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc/handlers"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/coin"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/config"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/devices/bitbox"
	bitboxHandlers "github.com/digitalbitbox/bitbox-wallet-app/backend/devices/bitbox/handlers"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/devices/device"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/keystore"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/signing"
	utilConfig "github.com/digitalbitbox/bitbox-wallet-app/util/config"
	"github.com/digitalbitbox/bitbox-wallet-app/util/errp"
	"github.com/digitalbitbox/bitbox-wallet-app/util/jsonp"
	"github.com/digitalbitbox/bitbox-wallet-app/util/locker"
	"github.com/digitalbitbox/bitbox-wallet-app/util/logging"
	"github.com/digitalbitbox/bitbox-wallet-app/util/system"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/text/language"
)

// Backend models the API of the backend.
type Backend interface {
	Config() *config.Config
	DefaultAppConfig() config.AppConfig
	Coin(string) (coin.Coin, error)
	AccountsStatus() string
	Testing() bool
	Accounts() []accounts.Interface
	CreateAndAddAccount(
		coin coin.Coin,
		code string,
		name string,
		getSigningConfiguration func() (*signing.Configuration, error),
	)
	UserLanguage() language.Tag
	OnAccountInit(f func(accounts.Interface))
	OnAccountUninit(f func(accounts.Interface))
	OnDeviceInit(f func(device.Interface))
	OnDeviceUninit(f func(deviceID string))
	DevicesRegistered() map[string]device.Interface
	Start() <-chan interface{}
	RegisterKeystore(keystore.Keystore)
	DeregisterKeystore()
	Register(device device.Interface) error
	Deregister(deviceID string)
	Rates() map[string]map[string]float64
	DownloadCert(string) (string, error)
	CheckElectrumServer(string, string) error
	RegisterTestKeystore(string)
}

// Handlers provides a web api to the backend.
type Handlers struct {
	Router  *mux.Router
	backend Backend
	// apiData consists of the port on which this API will run and the authorization token, generated by the
	// backend to secure the API call. The data is fed into the static javascript app
	// that is served, so the client knows where and how to connect to.
	apiData           *ConnectionData
	backendEvents     <-chan interface{}
	websocketUpgrader websocket.Upgrader
	log               *logrus.Entry
}

// ConnectionData contains the port and authorization token for communication with the backend.
type ConnectionData struct {
	port    int
	token   string
	devMode bool
}

// NewConnectionData creates a connection data struct which holds the port and token for the API.
// If the port is -1 or the token is empty, we assume dev-mode.
func NewConnectionData(port int, token string) *ConnectionData {
	return &ConnectionData{
		port:    port,
		token:   token,
		devMode: len(token) == 0,
	}
}

func (connectionData *ConnectionData) isDev() bool {
	return connectionData.port == -1 || connectionData.token == ""
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(
	backend Backend,
	connData *ConnectionData,
) *Handlers {
	log := logging.Get().WithGroup("handlers")
	router := mux.NewRouter()

	handlers := &Handlers{
		Router:  router,
		backend: backend,
		apiData: connData,
		websocketUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		log: logging.Get().WithGroup("handlers"),
	}

	getAPIRouter := func(subrouter *mux.Router) func(string, func(*http.Request) (interface{}, error)) *mux.Route {
		return func(path string, f func(*http.Request) (interface{}, error)) *mux.Route {
			return subrouter.Handle(path, ensureAPITokenValid(handlers.apiMiddleware(connData.isDev(), f),
				connData, log))
		}
	}

	apiRouter := router.PathPrefix("/api").Subrouter()
	getAPIRouter(apiRouter)("/qr", handlers.getQRCodeHandler).Methods("GET")
	getAPIRouter(apiRouter)("/config", handlers.getAppConfigHandler).Methods("GET")
	getAPIRouter(apiRouter)("/config/default", handlers.getDefaultConfigHandler).Methods("GET")
	getAPIRouter(apiRouter)("/config", handlers.postAppConfigHandler).Methods("POST")
	getAPIRouter(apiRouter)("/open", handlers.postOpenHandler).Methods("POST")
	getAPIRouter(apiRouter)("/update", handlers.getUpdateHandler).Methods("GET")
	getAPIRouter(apiRouter)("/version", handlers.getVersionHandler).Methods("GET")
	getAPIRouter(apiRouter)("/testing", handlers.getTestingHandler).Methods("GET")
	getAPIRouter(apiRouter)("/account-add", handlers.postAddAccountHandler).Methods("POST")
	getAPIRouter(apiRouter)("/accounts", handlers.getAccountsHandler).Methods("GET")
	getAPIRouter(apiRouter)("/accounts-status", handlers.getAccountsStatusHandler).Methods("GET")
	getAPIRouter(apiRouter)("/test/register", handlers.postRegisterTestKeystoreHandler).Methods("POST")
	getAPIRouter(apiRouter)("/test/deregister", handlers.postDeregisterTestKeystoreHandler).Methods("POST")
	getAPIRouter(apiRouter)("/rates", handlers.getRatesHandler).Methods("GET")
	getAPIRouter(apiRouter)("/coins/convertToFiat", handlers.getConvertToFiatHandler).Methods("GET")
	getAPIRouter(apiRouter)("/coins/convertFromFiat", handlers.getConvertFromFiatHandler).Methods("GET")
	getAPIRouter(apiRouter)("/coins/tltc/headers/status", handlers.getHeadersStatus("tltc")).Methods("GET")
	getAPIRouter(apiRouter)("/coins/tbtc/headers/status", handlers.getHeadersStatus("tbtc")).Methods("GET")
	getAPIRouter(apiRouter)("/coins/ltc/headers/status", handlers.getHeadersStatus("ltc")).Methods("GET")
	getAPIRouter(apiRouter)("/coins/btc/headers/status", handlers.getHeadersStatus("btc")).Methods("GET")
	getAPIRouter(apiRouter)("/certs/download", handlers.postCertsDownloadHandler).Methods("POST")
	getAPIRouter(apiRouter)("/certs/check", handlers.postCertsCheckHandler).Methods("POST")

	devicesRouter := getAPIRouter(apiRouter.PathPrefix("/devices").Subrouter())
	devicesRouter("/registered", handlers.getDevicesRegisteredHandler).Methods("GET")

	handlersMapLock := locker.Locker{}

	accountHandlersMap := map[string]*accountHandlers.Handlers{}
	getAccountHandlers := func(accountCode string) *accountHandlers.Handlers {
		defer handlersMapLock.Lock()()
		if _, ok := accountHandlersMap[accountCode]; !ok {
			accountHandlersMap[accountCode] = accountHandlers.NewHandlers(getAPIRouter(
				apiRouter.PathPrefix(fmt.Sprintf("/account/%s", accountCode)).Subrouter(),
			), log)
		}
		accHandlers := accountHandlersMap[accountCode]
		log.WithField("account-handlers", accHandlers).Debug("Account handlers")
		return accHandlers
	}

	backend.OnAccountInit(func(account accounts.Interface) {
		log.WithField("code", account.Code()).Debug("Initializing account")
		getAccountHandlers(account.Code()).Init(account)
	})
	backend.OnAccountUninit(func(account accounts.Interface) {
		getAccountHandlers(account.Code()).Uninit()
	})

	deviceHandlersMap := map[string]*bitboxHandlers.Handlers{}
	getDeviceHandlers := func(deviceID string) *bitboxHandlers.Handlers {
		defer handlersMapLock.Lock()()
		if _, ok := deviceHandlersMap[deviceID]; !ok {
			deviceHandlersMap[deviceID] = bitboxHandlers.NewHandlers(getAPIRouter(
				apiRouter.PathPrefix(fmt.Sprintf("/devices/%s", deviceID)).Subrouter(),
			), log)
		}
		return deviceHandlersMap[deviceID]
	}
	backend.OnDeviceInit(func(device device.Interface) {
		switch specificDevice := device.(type) {
		case *bitbox.Device:
			getDeviceHandlers(device.Identifier()).Init(specificDevice)
		}
	})
	backend.OnDeviceUninit(func(deviceID string) {
		getDeviceHandlers(deviceID).Uninit()
	})

	apiRouter.HandleFunc("/events", handlers.eventsHandler)

	handlers.backendEvents = backend.Start()

	return handlers
}

func writeJSON(w io.Writer, value interface{}) {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}

func (handlers *Handlers) getQRCodeHandler(r *http.Request) (interface{}, error) {
	data := r.URL.Query().Get("data")
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		return nil, errp.WithStack(err)
	}
	bytes, err := qr.PNG(256)
	if err != nil {
		return nil, errp.WithStack(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(bytes), nil
}

func (handlers *Handlers) getAppConfigHandler(_ *http.Request) (interface{}, error) {
	return handlers.backend.Config().AppConfig(), nil
}

func (handlers *Handlers) getDefaultConfigHandler(_ *http.Request) (interface{}, error) {
	return handlers.backend.DefaultAppConfig(), nil
}

func (handlers *Handlers) postAppConfigHandler(r *http.Request) (interface{}, error) {
	appConfig := config.AppConfig{}
	if err := json.NewDecoder(r.Body).Decode(&appConfig); err != nil {
		return nil, errp.WithStack(err)
	}
	return nil, handlers.backend.Config().SetAppConfig(appConfig)
}

func (handlers *Handlers) postOpenHandler(r *http.Request) (interface{}, error) {
	var url string
	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		return nil, errp.WithStack(err)
	}

	blocked := true

	for _, whitelistedURL := range []string{
		"https://shiftcrypto.ch/contact",
		"https://shiftcrypto.ch/shop",
		"https://shiftcrypto.ch/backup",
		"https://www.cryptocompare.com",
		"https://bitcoincore.org/en/2016/01/26/segwit-benefits/",
		"https://en.bitcoin.it/wiki/Bech32_adoption",
	} {
		if url == whitelistedURL {
			blocked = false
			break
		}
	}

	whitelistedPatterns := []string{
		"^https://blockstream\\.info/(testnet/)?tx/",
		"^http://explorer\\.litecointools\\.com/tx/",
		"^https://insight\\.litecore\\.io/tx/",
		"^https://etherscan\\.io/tx/",
		"^https://rinkeby\\.etherscan\\.io/tx/",
		"^https://ropsten\\.etherscan\\.io/tx/",
	}

	// Whitelist csv export.
	downloadDir, err := utilConfig.DownloadsDir()
	if err != nil {
		return nil, err
	}
	whitelistedPatterns = append(whitelistedPatterns,
		fmt.Sprintf("^%s", regexp.QuoteMeta(downloadDir)),
	)

	for _, pattern := range whitelistedPatterns {
		if regexp.MustCompile(pattern).MatchString(url) {
			blocked = false
			break
		}
	}
	if blocked {
		return nil, errp.Newf("Blocked /open with url: %s", url)
	}
	return nil, system.Open(url)
}

func (handlers *Handlers) getUpdateHandler(_ *http.Request) (interface{}, error) {
	return backend.CheckForUpdateIgnoringErrors(), nil
}

func (handlers *Handlers) getVersionHandler(_ *http.Request) (interface{}, error) {
	return backend.Version.String(), nil
}

func (handlers *Handlers) getTestingHandler(_ *http.Request) (interface{}, error) {
	return handlers.backend.Testing(), nil
}

func (handlers *Handlers) postAddAccountHandler(r *http.Request) (interface{}, error) {
	jsonBody := map[string]string{}
	if err := json.NewDecoder(r.Body).Decode(&jsonBody); err != nil {
		return nil, errp.WithStack(err)
	}
	// The following parameters only work for watch-only singlesig accounts at the moment.
	jsonCoinCode := jsonBody["coinCode"]
	jsonScriptType := jsonBody["scriptType"]
	jsonAccountName := jsonBody["accountName"]
	jsonExtendedPublicKey := jsonBody["extendedPublicKey"]

	coin, err := handlers.backend.Coin(jsonCoinCode)
	if err != nil {
		return nil, err
	}
	scriptType, err := signing.DecodeScriptType(jsonScriptType)
	if err != nil {
		return nil, err
	}
	keypath := signing.NewEmptyAbsoluteKeypath()
	extendedPublicKey, err := hdkeychain.NewKeyFromString(jsonExtendedPublicKey)
	if err != nil {
		return map[string]interface{}{"success": false, "errorCode": "xpubInvalid"}, nil
	}
	if extendedPublicKey.IsPrivate() {
		return map[string]interface{}{"success": false, "errorCode": "xprivEntered"}, nil
	}
	if btcCoin, ok := coin.(*btc.Coin); ok {
		expectedNet := &chaincfg.Params{
			HDPublicKeyID: btc.XPubVersionForScriptType(btcCoin, scriptType),
		}
		if !extendedPublicKey.IsForNet(expectedNet) {
			return map[string]interface{}{"success": false, "errorCode": "xpubWrongNet"}, nil
		}
	}
	configuration := signing.NewSinglesigConfiguration(scriptType, keypath, extendedPublicKey)
	getSigningConfiguration := func() (*signing.Configuration, error) {
		return configuration, nil
	}
	accountCode := fmt.Sprintf("%s-%s", configuration.Hash(), coin.Code())
	handlers.backend.CreateAndAddAccount(coin, accountCode, jsonAccountName, getSigningConfiguration)
	return map[string]interface{}{"success": true, "accountCode": accountCode}, nil
}

func (handlers *Handlers) getAccountsHandler(_ *http.Request) (interface{}, error) {
	type accountJSON struct {
		CoinCode              string `json:"coinCode"`
		Code                  string `json:"code"`
		Name                  string `json:"name"`
		BlockExplorerTxPrefix string `json:"blockExplorerTxPrefix"`
	}
	accounts := []*accountJSON{}
	for _, account := range handlers.backend.Accounts() {
		accounts = append(accounts, &accountJSON{
			CoinCode:              account.Coin().Code(),
			Code:                  account.Code(),
			Name:                  account.Name(),
			BlockExplorerTxPrefix: account.Coin().BlockExplorerTransactionURLPrefix(),
		})
	}
	return accounts, nil
}

func (handlers *Handlers) getAccountsStatusHandler(_ *http.Request) (interface{}, error) {
	return handlers.backend.AccountsStatus(), nil
}

func (handlers *Handlers) getDevicesRegisteredHandler(_ *http.Request) (interface{}, error) {
	jsonDevices := map[string]string{}
	for deviceID, device := range handlers.backend.DevicesRegistered() {
		jsonDevices[deviceID] = device.ProductName()
	}
	return jsonDevices, nil
}

func (handlers *Handlers) postRegisterTestKeystoreHandler(r *http.Request) (interface{}, error) {
	if !handlers.backend.Testing() {
		return nil, errp.New("Test keystore not available")
	}
	jsonBody := map[string]string{}
	if err := json.NewDecoder(r.Body).Decode(&jsonBody); err != nil {
		return nil, errp.WithStack(err)
	}
	pin := jsonBody["pin"]
	handlers.backend.RegisterTestKeystore(pin)
	return true, nil
}

func (handlers *Handlers) postDeregisterTestKeystoreHandler(_ *http.Request) (interface{}, error) {
	handlers.backend.DeregisterKeystore()
	return true, nil
}

func (handlers *Handlers) getRatesHandler(_ *http.Request) (interface{}, error) {
	return handlers.backend.Rates(), nil
}

func (handlers *Handlers) getConvertToFiatHandler(r *http.Request) (interface{}, error) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	amount := r.URL.Query().Get("amount")
	amountAsFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"errMsg":  "invalid amount",
		}, nil
	}
	rate := handlers.backend.Rates()[from][to]
	return map[string]interface{}{
		"success":    true,
		"fiatAmount": strconv.FormatFloat(amountAsFloat*rate, 'f', 2, 64),
	}, nil
}

func (handlers *Handlers) getConvertFromFiatHandler(r *http.Request) (interface{}, error) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	amount := r.URL.Query().Get("amount")
	amountAsFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"errMsg":  "invalid amount",
		}, nil
	}
	rate := handlers.backend.Rates()[to][from]
	result := 0.0
	if rate != 0.0 {
		result = amountAsFloat / rate
	}
	return map[string]interface{}{
		"success": true,
		"amount":  strconv.FormatFloat(result, 'f', 8, 64),
	}, nil
}

func (handlers *Handlers) getHeadersStatus(coinCode string) func(*http.Request) (interface{}, error) {
	return func(_ *http.Request) (interface{}, error) {
		coin, err := handlers.backend.Coin(coinCode)
		if err != nil {
			return nil, err
		}
		return coin.(*btc.Coin).Headers().Status()
	}
}

func (handlers *Handlers) postCertsDownloadHandler(r *http.Request) (interface{}, error) {
	var server string
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		return nil, errp.WithStack(err)
	}
	pemCert, err := handlers.backend.DownloadCert(server)
	if err != nil {
		return map[string]interface{}{
			"success":      false,
			"errorMessage": err.Error(),
		}, nil
	}
	return map[string]interface{}{
		"success": true,
		"pemCert": pemCert,
	}, nil
}

func (handlers *Handlers) postCertsCheckHandler(r *http.Request) (interface{}, error) {
	var server struct {
		Server  string `json:"server"`
		PEMCert string `json:"pemCert"`
	}
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		return nil, errp.WithStack(err)
	}

	if err := handlers.backend.CheckElectrumServer(
		server.Server,
		server.PEMCert); err != nil {
		return map[string]interface{}{
			"success":      false,
			"errorMessage": err.Error(),
		}, nil
	}
	return map[string]interface{}{
		"success": true,
	}, nil
}

func (handlers *Handlers) eventsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := handlers.websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}

	sendChan, quitChan := runWebsocket(conn, handlers.apiData, handlers.log)
	go func() {
		for {
			select {
			case <-quitChan:
				return
			default:
				select {
				case <-quitChan:
					return
				case event := <-handlers.backendEvents:
					sendChan <- jsonp.MustMarshal(event)
				}
			}
		}
	}()
}

// isAPITokenValid checks whether we are in dev or prod mode and, if we are in prod mode, verifies
// that an authorization token is received as an HTTP Authorization header and that it is valid.
func isAPITokenValid(w http.ResponseWriter, r *http.Request, apiData *ConnectionData, log *logrus.Entry) bool {
	methodLogEntry := log.WithField("path", r.URL.Path)
	// In dev mode, we allow unauthorized requests
	if apiData.devMode {
		// methodLogEntry.Debug("Allowing access without authorization token in dev mode")
		return true
	}
	methodLogEntry.Debug("Checking API token")

	if len(r.Header.Get("Authorization")) == 0 {
		methodLogEntry.Error("Missing token in API request. WARNING: this could be an attack on the API")
		http.Error(w, "missing token "+r.URL.Path, http.StatusUnauthorized)
		return false
	} else if len(r.Header.Get("Authorization")) != 0 && r.Header.Get("Authorization") != "Basic "+apiData.token {
		methodLogEntry.Error("Incorrect token in API request. WARNING: this could be an attack on the API")
		http.Error(w, "incorrect token", http.StatusUnauthorized)
		return false
	}
	return true
}

// ensureAPITokenValid wraps the given handler with another handler function that calls isAPITokenValid().
func ensureAPITokenValid(h http.Handler, apiData *ConnectionData, log *logrus.Entry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPITokenValid(w, r, apiData, log) {
			h.ServeHTTP(w, r)
		}
	})
}

func (handlers *Handlers) apiMiddleware(devMode bool, h func(*http.Request) (interface{}, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// recover from all panics and log error before panicking again
			if r := recover(); r != nil {
				handlers.log.WithField("panic", true).Errorf("%v\n%s", r, string(debug.Stack()))
				writeJSON(w, map[string]string{"error": fmt.Sprintf("%v", r)})
			}
		}()

		w.Header().Set("Content-Type", "text/json")
		if devMode {
			// This enables us to run a server on a different port serving just the UI, while still
			// allowing it to access the API.
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		}
		value, err := h(r)
		if err != nil {
			handlers.log.WithError(err).Error("endpoint failed")
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, value)
	})
}
