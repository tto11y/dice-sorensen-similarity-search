package logging

import (
	"fmt"
	"log/slog"
	"sync"
)

var ProgressMapMutex sync.RWMutex
var currentProgressIds = make(map[string]string)

// GetLogType creates a slice which can be used for logging
// it takes 3 arguments: subtype, contextId1 and correlationId/progressName
// this is then used to push logs with those parameters to loki
func GetLogType(logType ...string) []any {
	var temp []interface{}
	for i := 0; i < len(logType); i++ {
		if i == 0 {
			temp = append(temp, "subType")
		} else if i == 1 {
			temp = append(temp, "contextId1")
		} else if i == 2 {
			if len(logType[i]) <= 0 {
				break
			}
			ProgressMapMutex.Lock()
			if val, ok := currentProgressIds[logType[i]]; ok {
				// progressName found in memory, use that id
				temp = append(temp, "correlationId")
				temp = append(temp, val)
				ProgressMapMutex.Unlock()
				continue
			}
			ProgressMapMutex.Unlock()

			// id was supplied, use the supplied id
			temp = append(temp, "correlationId")
		} else {
			slog.Warn(fmt.Sprintf("getLogType: 4th parameter unknown: %v", logType[i]))
			break
		}
		temp = append(temp, logType[i])
	}
	if len(temp) <= 4 {
		// get current progress id if id was not specified
		ProgressMapMutex.Lock()
		if val, ok := currentProgressIds[logType[0]]; ok {
			temp = append(temp, "correlationId")
			temp = append(temp, val)
		}
		ProgressMapMutex.Unlock()
	}
	return temp
}

func GetLogTypeInitialization() []any {
	return GetLogType("initialization")
}

func GetLogTypeIntervalTask() []interface{} {
	return GetLogType("intervaltask")
}
