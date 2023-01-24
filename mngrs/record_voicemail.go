package mngrs

import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"

	"fmt"

	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/helpers"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type RecordVoicemailManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func NewRecordVoicemailManager(mngrCtx *types.Context, flow *types.Flow) *RecordVoicemailManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := RecordVoicemailManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}
func (man *RecordVoicemailManager) StartProcessing() {
	utils.Log(logrus.DebugLevel, "Creating bridge... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	channel := cell.CellChannel
	user := flow.User
	data := cell.Model.Data
	trimData, ok := data["trim"].(types.ModelDataBool)
	trim := false

	if ok {
		trim = trimData.Value
	}
	recording := helpers.NewRecording(user, nil, trim)
	_, err := recording.InitiateRecordingForChannel(channel)
	if err != nil {
		fmt.Println("recording err " + err.Error())
		return
	}
}
