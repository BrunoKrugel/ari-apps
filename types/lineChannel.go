package types

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/CyCoreSystems/ari/v5"
)

type LineChannel struct {
	LineBridge       *LineBridge
	Channel          *ari.ChannelHandle
	currentCellIndex int
	dtmfPressed      string
}

func (channel *LineChannel) RemoveFromBridge() {

}

func (channel *LineChannel) SafeHangup() error {
	if channel.Channel != nil {
		return channel.Channel.Hangup()
	}
	return errors.New("no Channel is existed")
}

func (channel *LineChannel) Answer() error {
	if channel.Channel != nil {
		channel.Channel.Answer()
		return nil
	}
	return errors.New("no Channel is existed")
}

func (channel *LineChannel) CreateCall(id string, params *CallParams) (*Call, error) {
	idAsInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	call := Call{
		CallId:  idAsInt,
		Channel: channel,
		Started: time.Now(),
		Params:  params}
	return &call, nil
}

func (channel *LineChannel) StartWaitingForRingTimeout(ctx *Context, noAnswer *Link, timeout int, wg *sync.WaitGroup, ringTimeoutChan <-chan bool, mode string) {
	duration := time.Now().Add(time.Duration(timeout) * time.Second)

	// Create a context that is both manually cancellable and will signal
	// a cancel at the specified duration.
	ringCtx, cancel := context.WithDeadline(context.Background(), duration)
	defer cancel()
	if mode == "dial" {
		wg.Done()
	}
	for {
		select {
		case <-ringTimeoutChan:
			fmt.Println("bridge in session. stopping ring timeout")
			return
		case <-ringCtx.Done():
			fmt.Println("Ring timeout elapsed.. ending all calls")
			if mode == "dial" {
				resp := ManagerResponse{
					Channel: channel,
					Link:    noAnswer}
				ctx.RecvChannel <- &resp
			} else {
				channel.SafeHangup()
			}
			return
		}
	}
}
