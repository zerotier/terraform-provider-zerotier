package zerotier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

// HostURL - Default Zerotier URL
const HostURL string = "https://my.zerotier.com/api"
const Token string = "D34DB33F"

// Client -
type Client struct {
	HostURL    string
	HTTPClient *http.Client
	Token      string
}

type Route struct {
	Target string  `json:"target"`
	Via    *string `json:"via"`
}

type IpRange struct {
	Start string `json:"ipRangeStart"`
	End   string `json:"ipRangeEnd"`
}

type V4AssignModeConfig struct {
	ZT bool `json:"zt"`
}

type Member struct {
	Id                 string        `json:"id"`
	NetworkId          string        `json:"networkId"`
	NodeId             string        `json:"nodeId"`
	OfflineNotifyDelay int           `json:"offlineNotifyDelay"`
	Name               string        `json:"name"`
	Description        string        `json:"description"`
	Hidden             bool          `json:"hidden"`
	Config             *MemberConfig `json:"config"`
}

type MemberConfig struct {
	Authorized      bool     `json:"authorized"`
	Capabilities    []int    `json:"capabilities"`
	Tags            [][]int  `json:"tags"`
	ActiveBridge    bool     `json:"activeBridge"`
	NoAutoAssignIps bool     `json:"noAutoAssignIps"`
	IpAssignments   []string `json:"ipAssignments"`
	CreationTime       int `json:"creationTime"`
	LastAuthorizedTime int `json:"lastAuthorizedTime"`
	VMajor             int `json:"vMajor"`
	VMinor             int `json:"vMinor"`
	VRev               int `json:"vRev"`
	VProto             int `json:"vProto"`
}

type Network struct {
	AuthorizedMemberCount int                    `json:"authorizedMemberCount"`
	CapabilitiesByName    map[string]interface{} `json:"capabilitiesByName"`
	Clock                 int                    `json:"clock"`
	Config                *NetworkConfig         `json:"config"`
	Description           string                 `json:"description"`
	Id                    string                 `json:"id"`
	OnlineMemberCount     int                    `json:"onlineMemberCount"`
	OwnerId               string                 `json:"ownerId"`
	Permissions           map[string]interface{} `json:"permissions"`
	RulesSource           string                 `json:"rulesSource"`
	Tags                  map[string]interface{} `json:"tags"`
	TagsByName            map[string]interface{} `json:"tagsByName"`
	TotalMemberCount      int                    `json:"totalMemberCount"`
	Type                  string                 `json:"type"`
	Ui                    map[string]interface{} `json:"ui"`
}

type NetworkConfig struct {
	Capabilities      []Capability       `json:"capabilities"`
	CreationTime      int64              `json:"creationTime"`
	IpAssignmentPools []IpRange          `json:"ipAssignmentPools"`
	LastModified      int64              `json:"lastModified"`
	Name              string             `json:"name"`
	Private           bool               `json:"private"`
	Revision          int                `json:"revision"`
	Routes            []Route            `json:"routes"`
	Rules             []IRule            `json:"rules"`
	Tags              []Tag              `json:"tags"`
	V4AssignMode      V4AssignModeConfig `json:"v4AssignMode"`
}

type Capability struct {
	Id      int     `json:"id"`
	Default bool    `json:"default"`
	Rules   []IRule `json:"rules"`
}

type Tag struct {
	Id      int  `json:"id"`
	Default *int `json:"default"`
}

type IRule interface {
	// default unmarshaljson just makes a
	// map[string]interface{} from { type: "ACTION_DROP" } etc
}

type TagByName struct {
	Tag
	Enums map[string]int `json:"enums"`
	Flags map[string]int `json:"flags"`
}

func NewClient(zerotier_controller_url, zerotier_controller_token *string) (*Client, error) {
	c := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		HostURL:    HostURL,
		Token:      Token,
	}

	if zerotier_controller_url != nil {
		c.HostURL = *zerotier_controller_url
	}

	if zerotier_controller_token != nil {
		c.Token = *zerotier_controller_token
	}

	return &c, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}

func (c *Client) GetNetwork(id string) (*Network, error) {
	url := fmt.Sprintf(c.HostURL+"/network/%s", id)
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var data Network
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteNetwork - Deletes an network
func (c *Client) DeleteNetwork(networkID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/network/%s", c.HostURL, networkID), nil)
	if err != nil {
		return err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return err
	}

	if string(body) != "" {
		return errors.New(string(body))
	}

	return nil
}

func (c *Client) UpdateNetwork(id string, network *Network) (*Network, error) {
	return c.postNetwork(id, network)
}

func (c *Client) postNetwork(id string, network *Network) (*Network, error) {
	url := strings.TrimSuffix(fmt.Sprintf(c.HostURL+"/network/%s", id), "/")

	// strip carriage returns?
	// network.RulesSource = strings.Replace(network.RulesSource, "\r", "", -1)
	j, err := json.Marshal(network)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var data Network
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func CIDRToRange(cidr string) (net.IP, net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil, err
	}
	first := ip.Mask(ipnet.Mask)
	last := make(net.IP, 4)
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		copy(last, ip)
	}
	// mirror what ZT console does
	// there must be a reason
	if first[3] == 0 {
		first[3] = 1
	}
	if last[3] == 255 {
		last[3] = 254
	}
	return first, last, nil

}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func (c *Client) CreateNetwork(network *Network) (*Network, error) {
	return c.postNetwork("", network)
}

//
// Member
//

func (c *Client) GetMember(nwid string, nodeId string) (*Member, error) {
	url := fmt.Sprintf(c.HostURL+"/network/%s/member/%s", nwid, nodeId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var data Member

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (c *Client) PollMember(nwid string, nodeId string) (*Member, error) {
	member, err := c.GetMember(nwid, nodeId)
	if err != nil {
		return nil, err
	}
	return member, nil
}

func (c *Client) postMember(member *Member, reqName string) (*Member, error) {
	url := fmt.Sprintf(c.HostURL+"/network/%s/member/%s", member.NetworkId, member.NodeId)

	j, err := json.Marshal(member)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var data Member

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (c *Client) CreateMember(member *Member) (*Member, error) {
	return c.postMember(member, "CreateMember")
}

func (c *Client) UpdateMember(member *Member) (*Member, error) {
	return c.postMember(member, "UpdateMember")
}

func (c *Client) DeleteMember(member *Member) error {
	url := fmt.Sprintf(c.HostURL+"/network/%s/member/%s", member.NetworkId, member.NodeId)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
