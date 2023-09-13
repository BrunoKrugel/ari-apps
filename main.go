package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	helpers "github.com/Lineblocs/go-helpers"
	_ "github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	"github.com/CyCoreSystems/ari-proxy/v5/client"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"lineblocs.com/processor/api"
	"lineblocs.com/processor/grpc"
	"lineblocs.com/processor/internal/config"
	errors "lineblocs.com/processor/internal/error"
	"lineblocs.com/processor/mngrs"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	zaplog "lineblocs.com/processor/utils/log"
)

var bridge *ari.BridgeHandle

type bridgeManager struct {
	h *ari.BridgeHandle
}

func main() {

	if err := zaplog.InitGlobalLogger(zap.NewProductionConfig()); err != nil {
		panic(err.Error())
	}

	cfg := config.NewConfig()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	connectCtx, cancelConnect := context.WithCancel(context.Background())
	defer cancelConnect()

	// Create the ARI connection
	cl, err := createARIConnection(connectCtx, cfg)
	if err != nil {
		panic(err.Error())
	}
	defer cl.Close()

	// Start the GRPC listener in a goroutine
	go grpc.StartListener(cl)

	// Log the startup messages
	zaplog.InfoWithContext(context.Background(), "Connected to ARI")
	zaplog.InfoWithContext(context.Background(), "Starting listener app")
	zaplog.InfoWithContext(context.Background(), "Listening for new calls")

	// Subscribe to StasisStart events
	sub := cl.Bus().Subscribe(nil, "StasisStart")

	for {
		select {
		case e := <-sub.Events():
			if v, ok := e.(*ari.StasisStart); ok {
				zaplog.InfoWithContext(ctx, "Got stasis start channel "+v.Channel.ID)
				go startExecution(ctx, cl, v, cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID)))
			}
		case <-ctx.Done():
			return
		case <-connectCtx.Done():
			cl.Close()
			return
		}
	}
}

func createARIConnection(ctx context.Context, cfg *config.Config) (ari.Client, error) {

	ariURL := cfg.ARIURL
	wsURL := cfg.WSURL

	zaplog.InfoWithContext(ctx, "Connecting to:"+ariURL)

	if cfg.UseProxy {
		zaplog.DebugWithContext(ctx, "Using ARI proxy")
		return client.New(ctx, client.WithApplication(cfg.Application), client.WithURI(os.Getenv("NATSGW_URL")))
	}

	zaplog.InfoWithContext(ctx, "Connecting to ARI server")

	return native.Connect(&native.Options{
		Application:  cfg.Application,
		Username:     cfg.Username,
		Password:     cfg.Password,
		URL:          ariURL,
		WebsocketURL: wsURL,
	})
}

func createCall() (types.Call, error) {
	return types.Call{}, nil
}
func createCallDebit(user *types.User, call *types.Call, direction string) error {
	return nil
}
func attachChannelLifeCycleListeners(flow *types.Flow, channel *types.LineChannel, ctx context.Context, callChannel chan *types.Call) {

	endSub := channel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()

	call := &types.Call{}

	for {

		select {
		case <-ctx.Done():
			return
		case <-endSub.Events():
			zaplog.DebugWithContext(ctx, "received stasis end event")
			call.Ended = time.Now()
			body, err := json.Marshal(types.StatusParams{
				CallId: call.CallId,
				Ip:     utils.GetPublicIp(),
				Status: "ended",
			})
			if err != nil {
				zaplog.DebugWithContext(ctx, err.Error())
				continue
			}

			_, err = api.SendHttpRequest("/call/updateCall", body)
			if err != nil {
				zaplog.DebugWithContext(ctx, err.Error())
				continue
			}
			err = createCallDebit(flow.User, call, "incoming")
			if err != nil {
				zaplog.DebugWithContext(ctx, "HTTP error: "+err.Error())
				continue
			}

		case call = <-callChannel:
			zaplog.DebugWithContext(ctx, "received setup call")
			zaplog.DebugWithContext(ctx, "id is "+strconv.Itoa(call.CallId))
		}
	}
}

func attachDTMFListeners(channel *types.LineChannel, ctx context.Context) {
	dtmfSub := channel.Channel.Subscribe(ari.Events.ChannelDtmfReceived)
	defer dtmfSub.Cancel()

	select {
	case <-ctx.Done():
		return
	case <-dtmfSub.Events():
		zaplog.DebugWithContext(ctx, "received DTMF event")
	}
}

func processIncomingCall(cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, exten string, callerId string) {
	go attachDTMFListeners(lineChannel, ctx)
	callChannel := make(chan *types.Call)
	go attachChannelLifeCycleListeners(flow, lineChannel, ctx, callChannel)

	zaplog.DebugWithContext(ctx, "Processing incoming call")
	zaplog.DebugWithContext(ctx, "Exten is:"+exten)
	zaplog.DebugWithContext(ctx, "Caller ID is:"+callerId)

	callParams := types.CallParams{
		From:        callerId,
		To:          exten,
		Status:      "start",
		Direction:   "inbound",
		UserId:      flow.User.Id,
		WorkspaceId: flow.User.Workspace.Id,
		ChannelId:   lineChannel.Channel.ID(),
	}

	body, err := json.Marshal(callParams)
	if err != nil {
		zaplog.ErrorWithContext(ctx, "JSON error: "+err.Error())
		return
	}

	zaplog.DebugWithContext(ctx, "Creating call...")
	resp, err := api.SendHttpRequest("/call/createCall", body)
	if err != nil {
		zaplog.ErrorWithContext(ctx, "HTTP error: "+err.Error())
		return
	}

	id := resp.Headers.Get("x-call-id")
	zaplog.DebugWithContext(ctx, "Call ID is:"+id)
	idAsInt, err := strconv.Atoi(id)
	if err != nil {
		zaplog.ErrorWithContext(ctx, err.Error())
		return
	}

	call := types.Call{
		CallId:  idAsInt,
		Channel: lineChannel,
		Started: time.Now(),
		Params:  &callParams,
	}

	flow.RootCall = &call
	zaplog.InfoWithContext(ctx, "Answering call")
	lineChannel.Answer()

	vars := make(map[string]string)
	go mngrs.ProcessFlow(cl, ctx, flow, lineChannel, vars, flow.Cells[0])

	callChannel <- &call

	select {
	case <-ctx.Done():
		return
	}
}

func startExecution(ctx context.Context, cl ari.Client, event *ari.StasisStart, h *ari.ChannelHandle) {
	helpers.Log(logrus.InfoLevel, "running app"+" channel "+h.Key().ID)

	action := event.Args[0]
	exten := event.Args[1]
	vals := make(map[string]string)
	vals["number"] = exten

	helpers.Log(logrus.DebugLevel, "received action: "+action)
	helpers.Log(logrus.DebugLevel, "EXTEN: "+exten)

	switch action {
	case "h":
		fmt.Println("Received h handler - not processing")
	case "DID_DIAL":
		fmt.Println("Already dialed - not processing")
		return
	case "DID_DIAL_2":

		fmt.Println("Already dialed - not processing")
	case "INCOMING_SIP_TRUNK":
		//domain := data.Domain
		exten := event.Args[1]
		callerId := event.Args[2]
		trunkAddr := event.Args[3]
		lineChannel := types.LineChannel{
			Channel: h}
		lineChannel.Answer()

		resp, err := api.GetUserByDID(exten)
		helpers.Log(logrus.DebugLevel, "exten ="+exten)
		helpers.Log(logrus.DebugLevel, "caller ID ="+callerId)
		helpers.Log(logrus.DebugLevel, "trunk addr ="+trunkAddr)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)
		err = utils.ProcessSIPTrunkCall(cl, lineChannel.Channel.Key(), user, &lineChannel, callerId, exten, trunkAddr)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			return

		}

	case "INCOMING_CALL":
		body, err := api.SendGetRequest("/user/getDIDNumberData", vals)

		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}

		var data types.FlowDIDData
		var flowJson types.FlowVars
		err = json.Unmarshal([]byte(body), &data)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}

		if utils.CheckFreeTrial(data.Plan) {
			helpers.Log(logrus.ErrorLevel, "Ending call due to free trial")
			h.Hangup()
			helpers.Log(logrus.DebugLevel, fmt.Sprintf("msg = %s", errors.FREE_TRIAL_ENDED))
			return
		}
		err = json.Unmarshal([]byte(data.FlowJson), &flowJson)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}

		body, err = api.SendGetRequest("/user/getWorkspaceMacros", vals)

		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}
		var macros []*types.WorkspaceMacro
		err = json.Unmarshal([]byte(body), &macros)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}

		lineChannel := types.LineChannel{
			Channel: h}
		user := types.NewUser(data.CreatorId, data.WorkspaceId, data.WorkspaceName)
		flow := types.NewFlow(
			data.FlowId,
			user,
			&flowJson,
			&lineChannel,
			macros,
			cl)

		helpers.Log(logrus.DebugLevel, "processing action: "+action)

		callerId := event.Args[2]
		fmt.Printf("Starting stasis with extension: %s, caller id: %s", exten, callerId)
		go processIncomingCall(cl, ctx, flow, &lineChannel, exten, callerId)
	case "OUTGOING_PROXY_ENDPOINT":

		callerId := event.Args[2]
		domain := event.Args[3]

		lineChannel := types.LineChannel{
			Channel: h}

		helpers.Log(logrus.DebugLevel, "looking up domain: "+domain)
		resp, err := api.GetUserByDomain(domain)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, domain)
		fmt.Printf("Calling %s\r\n", exten)
		lineChannel.Answer()
		err = utils.EnsureBridge(cl, lineChannel.Channel.Key(), user, &lineChannel, callerId, exten, "extension", nil)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			return

		}

	case "OUTGOING_PROXY":
		callerId := event.Args[2]
		domain := event.Args[3]

		helpers.Log(logrus.DebugLevel, "channel key: "+h.Key().ID)

		lineChannel := types.LineChannel{
			Channel: h}
		resp, err := api.GetUserByDomain(domain)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, domain)

		callerInfo, err := api.GetCallerId(user.Workspace.Domain, callerId)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get caller id. error: "+err.Error())
			return
		}
		fmt.Printf("setup caller id: " + callerInfo.CallerId)
		lineChannel.Answer()
		err = utils.EnsureBridge(cl, lineChannel.Channel.Key(), user, &lineChannel, callerInfo.CallerId, exten, "pstn", nil)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			return

		}

	case "OUTGOING_PROXY_MEDIA":
		helpers.Log(logrus.InfoLevel, "media service call..")
	case "OUTGOING_TRUNK_CALL":
		callerId := event.Args[2]
		trunkSourceIp := event.Args[3]
		helpers.Log(logrus.DebugLevel, "channel key: "+h.Key().ID)

		lineChannel := types.LineChannel{
			Channel: h}
		resp, err := api.GetUserByTrunkSourceIp(trunkSourceIp)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, resp.WorkspaceName)
		fmt.Printf("setup caller id: " + callerId)
		lineChannel.Answer()
		headers := make([]string, 0)
		headers = append(headers, "X-Lineblocs-User-SIP-Trunk-Calling-PSTN: true")
		err = utils.EnsureBridge(cl, lineChannel.Channel.Key(), user, &lineChannel, callerId, exten, "pstn", &headers)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			return

		}

	default:
		helpers.Log(logrus.InfoLevel, "unknown call type...")
	}
}
