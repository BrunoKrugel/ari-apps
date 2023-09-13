package types

import (
	"context"
	"regexp"
	"strings"

	"github.com/CyCoreSystems/ari/v5"
)

type Context struct {
	Flow        *Flow
	Cell        *Cell
	Channel     *LineChannel
	Runner      *Runner
	Vars        *FlowVars
	Context     context.Context
	Client      ari.Client
	RecvChannel chan<- *ManagerResponse
}

func convertVariableValues(value string, lineFlow *Flow) string {
	rex := regexp.MustCompile(`\{\{[\w\d\.]+\}\}`)
	out := rex.FindAllString(value, -1)

	for _, match := range out {
		value = strings.ReplaceAll(value, match, "")
	}
	return value
}

func processInterpolation(i ModelData, lineFlow *Flow) {
	switch value := i.(type) {
	case ModelDataStr:
		value.Value = convertVariableValues(value.Value, lineFlow)
	case ModelDataObj:
		for k, v := range value.Value {
			value.Value[k] = convertVariableValues(v, lineFlow)
		}
	case ModelDataArr:
		for k, v := range value.Value {
			value.Value[k] = convertVariableValues(v, lineFlow)
		}
	}
}

func processAllInterpolations(data map[string]ModelData, lineFlow *Flow) {
	for key, val := range data {
		if !strings.HasSuffix(key, "_before_interpolations") {
			interpolatedKey := key + "_before_interpolations"
			if before, ok := data[interpolatedKey]; ok {
				processInterpolation(before, lineFlow)
			}
			processInterpolation(val, lineFlow)
		}
	}
}

func NewContext(cl ari.Client, ctx context.Context, recvChannel chan<- *ManagerResponse, flow *Flow, cell *Cell, runner *Runner, channel *LineChannel) *Context {
	processAllInterpolations(cell.Model.Data, flow)
	return &Context{Client: cl, Context: ctx, Channel: channel, Cell: cell, Flow: flow, Runner: runner, RecvChannel: recvChannel}
}
