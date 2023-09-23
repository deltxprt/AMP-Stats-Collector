package main

import (
	"bytes"
	"encoding/json"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type GetInstances struct {
	Result []struct {
		Id                 int    `json:"Id"`
		InstanceId         string `json:"InstanceId"`
		FriendlyName       string `json:"FriendlyName"`
		AvailableInstances []GetInstance
	} `json:"result"`
}

type GetInstance struct {
	InstanceID       string `json:"InstanceID"`
	TargetID         string `json:"TargetID"`
	InstanceName     string `json:"InstanceName"`
	FriendlyName     string `json:"FriendlyName"`
	Module           string `json:"Module"`
	InstalledVersion struct {
		Major         int `json:"Major"`
		Minor         int `json:"Minor"`
		Build         int `json:"Build"`
		Revision      int `json:"Revision"`
		MajorRevision int `json:"MajorRevision"`
		MinorRevision int `json:"MinorRevision"`
	} `json:"InstalledVersion"`
	IsHTTPS               bool          `json:"IsHTTPS"`
	IP                    string        `json:"IP"`
	Port                  int           `json:"Port"`
	Daemon                bool          `json:"Daemon"`
	DaemonAutostart       bool          `json:"DaemonAutostart"`
	ExcludeFromFirewall   bool          `json:"ExcludeFromFirewall"`
	Running               bool          `json:"Running"`
	AppState              int           `json:"AppState"`
	Tags                  []interface{} `json:"Tags"`
	DiskUsageMB           int           `json:"DiskUsageMB"`
	ReleaseStream         int           `json:"ReleaseStream"`
	ManagementMode        int           `json:"ManagementMode"`
	Suspended             bool          `json:"Suspended"`
	IsContainerInstance   bool          `json:"IsContainerInstance"`
	ContainerMemoryMB     int           `json:"ContainerMemoryMB"`
	ContainerMemoryPolicy int           `json:"ContainerMemoryPolicy"`
	ContainerCPUs         float64       `json:"ContainerCPUs"`
	Metrics               struct {
		CPUUsage struct {
			RawValue int    `json:"RawValue"`
			MaxValue int    `json:"MaxValue"`
			Percent  int    `json:"Percent"`
			Units    string `json:"Units"`
			Color    string `json:"Color"`
			Color2   string `json:"Color2"`
			Color3   string `json:"Color3"`
		} `json:"CPU Usage,omitempty"`
		MemoryUsage struct {
			RawValue int    `json:"RawValue"`
			MaxValue int    `json:"MaxValue"`
			Percent  int    `json:"Percent"`
			Units    string `json:"Units"`
			Color    string `json:"Color"`
			Color3   string `json:"Color3"`
		} `json:"Memory Usage,omitempty"`
		ActiveUsers struct {
			RawValue int    `json:"RawValue"`
			MaxValue int    `json:"MaxValue"`
			Percent  int    `json:"Percent"`
			Units    string `json:"Units"`
			Color    string `json:"Color"`
			Color3   string `json:"Color3"`
		} `json:"Active Users,omitempty"`
	} `json:"Metrics"`
	ApplicationEndpoints []struct {
		DisplayName string `json:"DisplayName"`
		Endpoint    string `json:"Endpoint"`
		Uri         string `json:"Uri"`
	} `json:"ApplicationEndpoints"`
	DisplayImageSource string `json:"DisplayImageSource,omitempty"`
	ModuleDisplayName  string `json:"ModuleDisplayName,omitempty"`
}

var (
	url          string
	username     string
	password     string
	sessionId    string
	influxAddr   string
	organization string
	bucket       string
	token        string
)

func ApiCall(url string, data map[string]string) ([]byte, error) {
	//fmt.Println(loginUrl)
	var body []byte
	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(data)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, payloadBuf)
	request.Header.Set("accept", "application/json; charset=UTF-8")

	if err != nil {
		return body, err
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return body, err
	}
	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)
	if err != nil {
		return body, err
	}
	//fmt.Println(string(body))
	return body, nil
}
func getSessionId() (string, error) {
	type Login struct {
		SessionID string `json:"sessionID"`
	}
	loginResult := Login{}

	loginUrl := url + "/API/Core/Login"

	data := map[string]string{
		"username":   username,
		"password":   password,
		"token":      "",
		"rememberMe": "false",
	}

	byteLogin, err := ApiCall(loginUrl, data)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(byteLogin, &loginResult)
	if err != nil {
		return "", err
	}
	return loginResult.SessionID, nil
}

func updateInstancesHandler() error {
	getInstancesUrl := url + "/API/ADSModule/GetInstances"

	var listInstances GetInstances

	dataBody := map[string]string{"SESSIONID": sessionId}

	bytesInstances, err := ApiCall(getInstancesUrl, dataBody)

	if err != nil {
		return err
	}

	err = json.Unmarshal(bytesInstances, &listInstances)
	if err != nil {
		return err
	}

	for _, host := range listInstances.Result {

		hostInstances := host.AvailableInstances
		slog.Info(host.FriendlyName)

		for _, instance := range hostInstances {

			if instance.InstanceName == "ADS01" {
				continue
			}
			err := sendStats(instance)
			if err != nil {
				return err
			}
			//log.Println(instanceID, instanceFriendlyName, instanceName)
		}
	}
	return nil
}

func sendStats(instance GetInstance) error {
	client := influxdb2.NewClient(influxAddr, token)

	writeAPI := client.WriteAPIBlocking(organization, bucket)

	point := write.NewPointWithMeasurement(instance.InstanceName).
		AddField("CPU_Usage", instance.Metrics.CPUUsage.RawValue).
		AddField("Memory_Usage", instance.Metrics.MemoryUsage.RawValue).
		AddField("Memory_Max", instance.Metrics.MemoryUsage.MaxValue).
		AddField("Users_Current", instance.Metrics.ActiveUsers.RawValue).
		AddField("Users_Max", instance.Metrics.ActiveUsers.MaxValue).
		AddField("Running", instance.Running).
		AddField("Module", instance.Module).
		SetTime(time.Now())
	//point := write.NewPoint(instance.InstanceName, nil, instanceInfo, time.Now())

	err := writeAPI.WritePoint(context.Background(), point)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/AMP-Stats/")
	viper.AddConfigPath("/config/")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Error("unable to read config file", err)
	}
	url = viper.GetString("url")
	username = viper.GetString("username")
	password = viper.GetString("password")
	influxAddr = viper.GetString("influxAddr")
	organization = viper.GetString("org")
	bucket = viper.GetString("bucket")
	token = viper.GetString("token")
	for {
		logger.Info("Updating instances")
		sessionId, err = getSessionId()
		if err != nil {
			logger.Error("unable to get session id", err)
		}
		err = updateInstancesHandler()
		if err != nil {
			logger.Error("unable to update instances", err)
		}
		time.Sleep(time.Second * 10)
	}
}
