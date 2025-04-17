// Copyright 2025 AUTHORS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package catch

import (
	"log"

	"github.com/yeetrun/yeet/pkg/db"
	"github.com/yeetrun/yeet/pkg/svc"
)

type ServiceDataType string
type ComponentStatus string

const (
	ServiceDataTypeService ServiceDataType = "service"
	ServiceDataTypeCron    ServiceDataType = "cron"
	ServiceDataTypeDocker  ServiceDataType = "docker"
	ServiceDataTypeUnknown ServiceDataType = "unknown"

	ComponentStatusStarting ComponentStatus = "starting"
	ComponentStatusRunning  ComponentStatus = "running"
	ComponentStatusStopping ComponentStatus = "stopping"
	ComponentStatusStopped  ComponentStatus = "stopped"
	ComponentStatusUnknown  ComponentStatus = "unknown"
)

type ServiceStatusData struct {
	ServiceName     string                `json:"serviceName"`
	ServiceType     ServiceDataType       `json:"serviceType"`
	ComponentStatus []ComponentStatusData `json:"components"`
}

type ComponentStatusData struct {
	Name   string          `json:"name"`
	Status ComponentStatus `json:"status"`
}

func ComponentStatusFromServiceStatus(st svc.Status) ComponentStatus {
	switch st {
	case svc.StatusRunning:
		return ComponentStatusRunning
	case svc.StatusStopped:
		return ComponentStatusStopped
	case svc.StatusUnknown:
		return ComponentStatusUnknown
	default:
		log.Printf("unknown service status: %v", st)
		return ComponentStatusUnknown
	}
}

func ServiceDataTypeFromServiceType(st db.ServiceType) ServiceDataType {
	switch st {
	case db.ServiceTypeSystemd:
		return ServiceDataTypeService
	case db.ServiceTypeDockerCompose:
		return ServiceDataTypeDocker
	default:
		return ServiceDataTypeUnknown
	}
}

func ServiceDataTypeFromUnitType(unitType string) ServiceDataType {
	switch unitType {
	case "service":
		return ServiceDataTypeService
	case "cron":
		return ServiceDataTypeCron
	case "docker":
		return ServiceDataTypeDocker
	default:
		log.Printf("unknown unit type: %q", unitType)
		return ServiceDataTypeUnknown
	}
}
