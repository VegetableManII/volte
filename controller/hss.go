package controller

import (
	"context"
	"volte/common"
)

func ProcessHSSCore(ctx context.Context, coreIn, coreOut chan *common.Msg) {
	for {
		select {
		case m := <-coreIn:
			if m.Type == common.EPCPROTOCAL {

			}
		}
	}
}

func processorhss() {

}
